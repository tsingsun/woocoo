package oasgen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	intmap = map[string]code.Type{
		"int":    code.TypeInt,
		"int8":   code.TypeInt8,
		"int16":  code.TypeInt16,
		"int32":  code.TypeInt32,
		"int64":  code.TypeInt64,
		"uint":   code.TypeUint,
		"uint8":  code.TypeUint8,
		"uint16": code.TypeUint16,
		"uint32": code.TypeUint32,
		"uint64": code.TypeUint64,
	}
)

// SchemaOption helps gen schema
type SchemaOption struct {
	// Required indicates if the schema is required.
	Required bool
	// IsComponent indicates if the schema is a component schema.
	IsComponent bool
}

// Schema is for schema of openapi3
// a struct , the struct 's field all is Schema
// Properties are sort by name, because openapi3 use map to store properties,map is not ordered.
type Schema struct {
	SchemaOption
	Spec        *openapi3.SchemaRef // The original OpenAPIv3 Schema.
	Name        string              // The name of the schema.
	Type        *code.TypeInfo      // The type of the schema.
	IsRef       bool
	HasRegular  bool     // if schema has a pattern setting
	validations []string // the expression string for validator
	StructTags  []string
	properties  []*Schema
	Properties  map[string]*Schema
	IsInline    bool // if schema is inline , schema is embedded in another schema
	IsReplace   bool // if schema is replaced by model defined in config
	IsAlias     bool // if schema is alias of not go native type
	IsArray     bool
	// ItemSchema is the schema type of the array or map.
	ItemSchema *Schema
}

// GenSchemaType generates the type of the parameter by SPEC.
// schema type from : the schema's sepc , additionalProperties, array items
func (sch *Schema) GenSchemaType(c *Config, name string, spec *openapi3.SchemaRef) {
	var info *code.TypeInfo
	if spec == nil {
		spec = sch.Spec
	} else {
		sch.Spec = spec
	}
	if tm := c.TypeMap[spec.Ref]; tm != nil {
		sch.Type = tm.Clone()
		return
	}
	sv := spec.Value
	sepcType := sv.Type
	if sv.AllOf != nil {
		sepcType = "object"
	} else if sv.Type == "" && (sv.OneOf == nil || sv.AnyOf == nil) {
		// todo check oneOf and anyOf
		sepcType = "object"
	}

	switch sepcType {
	case "array":
		sch.IsArray = true
		info = &code.TypeInfo{
			Type:     code.TypeOther,
			Nillable: true,
			PkgPath:  c.Package,
			PkgName:  code.PkgShortName(c.Package),
		}
		subSch := Schema{}
		subSch.GenSchemaType(c, schemaNameFromRef(sv.Items.Ref), sv.Items)
		sch.ItemSchema = &subSch
		info.Ident = "[]" + subSch.Type.String()
		//  has RType
		if subSch.Type.RType != nil {
			iv := reflect.MakeSlice(subSch.Type.RType.ReflectType(), 0, 0)
			rt, err := code.ParseGoType(iv)
			if err != nil {
				panic(err)
			}
			info.Ident = rt.String()
			info.PkgPath = rt.PkgPath
			info.RType = rt
		}
		if subSch.Type.Type == code.TypeOther {
			if !strings.HasPrefix(subSch.Type.String(), "*") {
				info.Ident = "[]*" + subSch.Type.String() // make slice item is pointer to easy set value
			}
		}
	case "integer":
		tp, ok := intmap[sv.Format]
		if !ok {
			tp = code.TypeInt
		}
		info = &code.TypeInfo{Type: tp}
	case "number":
		switch sv.Format {
		case "double":
			info = &code.TypeInfo{Type: code.TypeFloat64}
		case "float":
			info = &code.TypeInfo{Type: code.TypeFloat32}
		default:
			info = &code.TypeInfo{Type: code.TypeFloat64}
		}
	case "boolean":
		info = &code.TypeInfo{Type: code.TypeBool}
	case "string":
		switch sv.Format {
		case "byte":
			info = &code.TypeInfo{Type: code.TypeBytes, Nillable: true}
		case "date":
			info = &code.TypeInfo{Type: code.TypeTime}
			// time use struct tag to set format
			sch.StructTags = append(sch.StructTags, fmt.Sprintf(`time_format:%q`, time.DateOnly))
		case "date-time":
			info = &code.TypeInfo{Type: code.TypeTime, PkgPath: "time"}
			sch.StructTags = append(sch.StructTags, fmt.Sprintf(`time_format:%q`, time.RFC3339))
		case "duration":
			info = &code.TypeInfo{Type: code.TypeUint64, PkgPath: "time"}
			info.Ident = "Duration"
		case "email":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.validations = append(sch.validations, "email")
		case "uuid":
			rt, err := code.ParseGoType(uuid.New())
			if err != nil {
				panic(err)
			}
			info = &code.TypeInfo{Type: code.TypeUUID, RType: rt, Ident: rt.Ident, PkgPath: rt.PkgPath, PkgName: code.PkgName(rt.String())}
		case "json":
			info = &code.TypeInfo{Type: code.TypeJSON, Nillable: true}
		case "hostname":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.validations = append(sch.validations, "hostname_rfc1123")
		case "ip":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.validations = append(sch.validations, "ip")
		case "ipv4":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.validations = append(sch.validations, "ipv4")
		case "ipv6":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.validations = append(sch.validations, "ipv6")
		case "uri":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.validations = append(sch.validations, "uri")
		case "binary":
			info = &code.TypeInfo{Type: code.TypeBytes, Nillable: true}
		default:
			if len(sv.Enum) != 0 {
				// if empty name (anonymous) , ident will be empty, and TypeEnum.String() will be string.
				info = &code.TypeInfo{Type: code.TypeEnum, Ident: helper.Pascal(name)}
			} else {
				info = &code.TypeInfo{Type: code.TypeString}
			}
		}
	case "binary":
		info = &code.TypeInfo{Type: code.TypeBytes, Nillable: true}
	case "object":
		info = &code.TypeInfo{Type: code.TypeOther}
		if sv.AdditionalProperties.Schema != nil {
			if sv.AdditionalProperties.Schema.Ref != "" {
				sch.GenSchemaType(c, schemaNameFromRef(sv.AdditionalProperties.Schema.Ref), sv.AdditionalProperties.Schema)
			} else {
				subSch := &Schema{}
				subSch.GenSchemaType(c, "", sv.AdditionalProperties.Schema)
				info.Ident = "map[string]" + subSch.Type.String()
				sch.ItemSchema = subSch
			}
			info.Nillable = true

			break
		} else if isJsonRawObject(c, name, sv) {
			info.Type = code.TypeJSON
			info.Nillable = true
			break
		}
		// inline object,generate in package path
		if spec.Ref != "" {
			info.Ident = schemaNameFromRef(spec.Ref)
		} else {
			info.Ident = helper.Pascal(name)
		}
		if info.Ident == "" {
			panic("object should be a ident name")
		}
		info.PkgPath = c.Package
		info.PkgName = code.PkgShortName(c.Package)
	case "":
		// TODO
		return
	default:
		panic(fmt.Errorf("unhandled OpenAPISchema type: %s", sv.Type))
	}

	if info == nil {
		return
	}

	if spec.Value.Nullable && !info.Nillable {
		if info.Ident != "" {
			info.Ident = "*" + info.Ident
		}
		info.Nillable = true
	}
	sch.Type = info
	return
}

var updateContentTypes = make(map[string]struct{})

// AppendContentTypeStructTag parse content type and append to struct tags.if depends on Request or Response content type.
// the name of schema ref will be Pascal format, but in openapi lower Camel format.
func (sch *Schema) AppendContentTypeStructTag(c *Config, tagName string, contentTypes []string) {
	for _, contentType := range contentTypes {
		switch contentType {
		case binding.MIMEJSON:
			if HasTag(sch.StructTags, "json") {
				continue
			}
			tagName = lowCamelFirst(tagName)
			if sch.Required {
				sch.StructTags = append(sch.StructTags, fmt.Sprintf(`json:"%s"`, tagName))
			} else {
				sch.StructTags = append(sch.StructTags, fmt.Sprintf(`json:"%s,omitempty"`, tagName))
			}
		case binding.MIMEPOSTForm:
			if HasTag(sch.StructTags, "form") {
				continue
			}
			sch.StructTags = append(sch.StructTags, fmt.Sprintf(`form:"%s"`, tagName))
		case binding.MIMEXML, binding.MIMEXML2:
			if HasTag(sch.StructTags, "xml") {
				continue
			}
			if x := sch.Spec.Value.XML; x != nil {
				tagName = ""
				if x.Prefix != "" {
					tagName = x.Prefix + ":"
				}
				tagName += x.Name
				if x.Attribute {
					tagName += ",attr"
				}
			}
			sch.StructTags = append(sch.StructTags, fmt.Sprintf(`xml:"%s"`, tagName))
		case binding.MIMEMSGPACK2:
			if HasTag(sch.StructTags, "msgpack") {
				continue
			}
			sch.StructTags = append(sch.StructTags, fmt.Sprintf(`msgpack:"%s"`, tagName))
		default:
			if HasTag(sch.StructTags, "form") {
				continue
			}
			sch.StructTags = append(sch.StructTags, fmt.Sprintf(`form:"%s"`, tagName))
		}
	}

	// update schema info
	if sch.IsRef || sch.IsArray {
		ref := sch.Spec.Ref
		if sch.IsObjectArray() {
			// array
			ref = sch.Spec.Value.Items.Ref
		}
		if s, ok := c.schemas[ref]; ok {
			var cts []string
			for _, contentType := range contentTypes {
				key := ref + ":" + contentType
				if _, ok := updateContentTypes[key]; ok {
					continue
				}
				cts = append(cts, contentType)
				updateContentTypes[key] = struct{}{}
			}
			if len(cts) > 0 {
				s.AppendContentTypeStructTag(c, s.Name, cts)
			}
		}
		//else if sch.Type.Type == code.TypeOther && !sch.Type.Nillable {
		//	// nillable like slice map, pointer do not need to be added to the schema,those are inline or has added to the schema
		//	c.AddSchema(sch)
		//}
	}
	for _, property := range sch.properties {
		property.AppendContentTypeStructTag(c, property.Name, contentTypes)
	}
}

// IsObjectArray returns true if the schema is an array of objects
func (sch *Schema) IsObjectArray() bool {
	return sch.IsArray
}

// StructString returns the string representation of the schema's struct type.
func (sch *Schema) StructString() string {
	s := sch.Type.String()
	if strings.HasPrefix(s, "*") {
		return s[1:]
	}
	return s
}

// TypeString returns the string representation of the schema's type.
// Response type required is false
func (sch *Schema) TypeString() string {
	s := sch.Type.String()

	if sch.Required {
		return sch.StructString()
	}
	if sch.Type.Nillable {
		return s
	}
	if !sch.Required && !sch.Type.Nillable {
		if strings.HasPrefix(s, "*") {
			return s
		}
		return "*" + s
	}
	if sch.Type.Ident != "" && !strings.HasPrefix(s, "*") {
		return "*" + s
	}
	return s
}

// StructTagsString return tag string when output in template.It will sort by asc.
func (sch *Schema) StructTagsString() string {
	if len(sch.StructTags) == 0 {
		return ""
	}
	return "`" + strings.Join(sortAsc(sch.StructTags), " ") + "`"
}

// CollectTags collects all struct tags from the schema and its children.
// validations see: https://pkg.go.dev/github.com/go-playground/validator/v10
func (sch *Schema) CollectTags() {
	specValue := sch.Spec.Value
	if specValue.Pattern != "" {
		sch.HasRegular = true
		rName := AddPattern(specValue.Pattern)
		pattenMap[specValue.Pattern] = rName
		tn := fmt.Sprintf("%s=%s", TagRegular, rName)
		sch.validations = append(sch.validations, tn)
	}
	if sch.Type.Numeric() {
		if specValue.Max != nil {
			op := "lte"
			if specValue.ExclusiveMax {
				op = "lt"
			}
			sch.validations = append(sch.validations, fmt.Sprintf("%s=%v", op, *specValue.Max))
		}
		if specValue.Min != nil {
			op := "gte"
			if specValue.ExclusiveMin {
				op = "gt"
			}
			sch.validations = append(sch.validations, fmt.Sprintf("%s=%v", op, *specValue.Min))
		}
	}
	if sch.Type.Stringer() {
		if specValue.MaxLength != nil {
			sch.validations = append(sch.validations, fmt.Sprintf("max=%d", *specValue.MaxLength))
		}
		if specValue.MinLength != 0 {
			sch.validations = append(sch.validations, fmt.Sprintf("min=%d", specValue.MinLength))
		}
	}
	if sch.Spec.Value.Type == "array" {
		if specValue.MaxItems != nil {
			sch.validations = append(sch.validations, fmt.Sprintf("max=%d", specValue.MaxItems))
		}
		if specValue.MinItems != 0 {
			sch.validations = append(sch.validations, fmt.Sprintf("min=%d", specValue.MinItems))
		}
		if specValue.UniqueItems {
			sch.validations = append(sch.validations, "unique")
		}
	}
	for k, v := range specValue.Extensions {
		switch k {
		case goTagValidator:
			sch.validations = append(sch.validations, v.(string))
		}
	}
	if sch.IsEnum() {
		if sch.IsArray {
			sch.validations = append(sch.validations, "dive")
		}
		sch.validations = append(sch.validations, "oneof="+strings.Join(sch.EnumValues(), " "))
	}
	// notice : required check must be last, it handles the omitempty which order is important.
	if sch.Required {
		sch.validations = append([]string{"required"}, sch.validations...)
	} else {
		// if not required, Add omitempty as needed, and omitempty must be the first
		if len(sch.validations) > 0 {
			sch.validations = append([]string{"omitempty"}, sch.validations...)
		}
	}
	if len(sch.validations) > 0 {
		sch.StructTags = append(sch.StructTags, fmt.Sprintf(`binding:"%s"`, strings.Join(sch.validations, ",")))
	}
}

// CopyTo copies the schema's base properties to the target schema
func (sch *Schema) CopyTo(tg *Schema) {
	tg.properties = sch.properties
	tg.Properties = sch.Properties
	tg.StructTags = sch.StructTags
	tg.validations = sch.validations
	tg.HasRegular = sch.HasRegular
	tg.IsReplace = sch.IsReplace
	tg.IsInline = sch.IsInline
	tg.IsAlias = sch.IsAlias
	tg.Type = sch.Type
	tg.IsArray = sch.IsArray
	tg.ItemSchema = sch.ItemSchema
	tg.IsComponent = sch.IsComponent
}

// FixRequired checks if the schema is required and updates the schema type if needed
func (sch *Schema) FixRequired() {
	if sch.Type.Type == code.TypeOther && !sch.Type.Nillable {
		if sch.Required {
			if strings.HasPrefix(sch.Type.Ident, "*") {
				sch.Type.Ident = sch.Type.Ident[1:]
			}
		} else {
			if !strings.HasPrefix(sch.Type.Ident, "*") {
				sch.Type.Ident = "*" + sch.Type.Ident
			}
		}
	}
}

// IsEnum returns true if the field is an enum Schema.
func (sch *Schema) IsEnum() bool {
	if sch.ItemSchema != nil {
		return sch.ItemSchema.IsEnum()
	}
	return sch.Type != nil && sch.Type.Type == code.TypeEnum
}

// EnumsProperties returns the enum properties of the schema, if any.
func (sch *Schema) EnumsProperties() []*Schema {
	var enums []*Schema
	if sch.IsEnum() {
		enums = append(enums, sch)
	}
	for _, property := range sch.properties {
		if property.IsEnum() {
			enums = append(enums, property)
		}
	}
	return enums
}

// EnumValues returns the enum values of the schema, if any. It only supports string enum.
func (sch *Schema) EnumValues() []string {
	var vs []string
	if sch.ItemSchema != nil {
		return sch.ItemSchema.EnumValues()
	}
	for _, e := range sch.Spec.Value.Enum {
		ev, ok := e.(string)
		if !ok {
			panic(fmt.Sprintf("enum only support string values:%s", sch.Name))
		}
		vs = append(vs, ev)
	}
	return vs
}

func genSchemaRef(c *Config, name string, spec *openapi3.SchemaRef, opts SchemaOption) *Schema {
	if spec.Ref != "" && name == "" { // if it's a ref, we need to get the name from the ref
		name = schemaNameFromRef(spec.Ref)
	}
	sc := &Schema{
		SchemaOption: opts,
		Name:         name,
		Spec:         spec,
		Properties:   make(map[string]*Schema),
		IsRef:        spec.Ref != "",
	}

	if sc.IsRef {
		if s, ok := c.schemas[spec.Ref]; ok {
			s.CopyTo(sc)
			// if it's an alias, we need to get the name for reference type
			if s.IsAlias {
				tt := *s.Type
				sc.Type = &tt
				sc.Type.Ident = helper.Pascal(s.Name)
			}

			sc.FixRequired()
			return sc
		}
	}
	sc.GenSchemaType(c, name, spec)
	sc.FixRequired()
	for k, v := range spec.Value.Extensions {
		switch k {
		case goTag:
			sc.StructTags = append(sc.StructTags, v.(string))
		}
	}

	// allOf
	if spec.Value != nil && len(spec.Value.AllOf) > 0 {
		// allof node is a new schema
		sc.IsRef = false
		for i, one := range spec.Value.AllOf {
			if one.Ref != "" {
				gs := genSchemaRef(c, "", one, SchemaOption{})
				// inline struct
				gs.IsInline = true
				sc.Properties[strconv.Itoa(i)] = gs
				sc.properties = append(sc.properties, gs)
			} else {
				for _, oname := range sortPropertyKeys(one.Value.Properties) {
					schemaRef := one.Value.Properties[oname]
					gs := genSchemaRef(c, oname, schemaRef, SchemaOption{Required: helper.InStrSlice(one.Value.Required, oname)})
					sc.Properties[oname] = gs
					sc.properties = append(sc.properties, gs)
				}
			}
		}
	}
	sc.CollectTags()
	sc.setAlias()
	// set to c.schemas , and avoid recursive
	if sc.IsRef {
		if sc.IsAlias {
			// not the root schema, set alias to type todo set type other field
			sc.Type.Ident = helper.Pascal(schemaNameFromRef(sc.Spec.Ref))
		}
	}
	sc.genProperties(c, spec)
	return sc
}

func (sch *Schema) genProperties(c *Config, spec *openapi3.SchemaRef) {
	for _, pname := range sortPropertyKeys(spec.Value.Properties) {
		schemaRef := sch.Spec.Value.Properties[pname]
		gs := genSchemaRef(c, pname, schemaRef,
			SchemaOption{
				Required: helper.InStrSlice(spec.Value.Required, pname),
			},
		)
		// if component schema is enum, we need to set the ident of the type to avoid conflict
		if gs.IsEnum() && sch.IsComponent {
			if gs.ItemSchema != nil {
				gs.ItemSchema.Type.Ident = helper.Pascal(sch.Name) + helper.Pascal(pname)
			} else {
				gs.Type.Ident = helper.Pascal(sch.Name) + helper.Pascal(pname)
			}
		}
		sch.Properties[pname] = gs
		sch.properties = append(sch.properties, gs)
	}
}

// SetAlias set IsAlias, it just calls in genComponentSchemas
func (sch *Schema) setAlias() {
	switch {
	// typejson mapped by json.RawMessage
	case sch.Type.Type == code.TypeOther:
		if sch.IsArray {
			sch.IsAlias = true
		}
		if strings.HasPrefix(sch.Type.Ident, "map[string]") {
			sch.IsAlias = true
		}
	case sch.Type.Type == code.TypeJSON:
		sch.IsAlias = true
	}
}

func genComponentSchemas(c *Config, spec *openapi3.T) {
	// copy c.TypeMap to tmpTypeMap
	// make sure this mothod is run first,because type map will change in genSchemaRef by request or response
	// and isReplace will be not correct
	tmpTypeMap := make(map[string]*code.TypeInfo)
	for k, v := range c.TypeMap {
		tmpTypeMap[k] = v
	}
	for _, name := range sortPropertyKeys(spec.Components.Schemas) {
		schemaRef := spec.Components.Schemas[name]
		k := "#/components/schemas/" + name
		gs := genSchemaRef(c, name, schemaRef, SchemaOption{IsComponent: true})
		if tp, ok := tmpTypeMap[k]; ok { // schemas do not has a ref
			gs.IsReplace = true
			gs.Type = tp
		}
		//gs.setAlias()
		c.AddTypeMap(k, gs.Type)
		c.AddSchema(k, gs)
	}
	return
}

func isTypeMap(c *Config, name string) bool {
	k := "#/components/schemas/" + name
	_, ok := c.TypeMap[k]
	return ok
}

func isJsonRawObject(c *Config, name string, schema *openapi3.Schema) bool {
	if isTypeMap(c, name) {
		return false
	}
	if schema.Type != "object" {
		return false
	}
	if len(schema.Properties) != 0 {
		return false
	}
	if len(schema.AllOf) != 0 {
		return false
	}
	if len(schema.AnyOf) != 0 {
		return false
	}
	if len(schema.OneOf) != 0 {
		return false
	}
	if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
		return false
	}
	return true
}
