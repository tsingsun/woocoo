package oasgen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"reflect"
	"strings"
	"time"
)

const (
	ComponentsRefPrefix = "#/components/schemas/"
	RequestPrefix       = "#/components/requestBodies/"
	ResponsePrefix      = "#/components/response/"
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

// SchemaZone indicates the zone of the schema.
type SchemaZone int

const (
	// SchemaZoneComponent indicates the schema is a component schema.
	SchemaZoneComponent SchemaZone = iota
	// SchemaZoneRequest indicates the schema is in request parameters.
	SchemaZoneRequest
	// SchemaZoneResponse indicates the schema is in response.
	SchemaZoneResponse
)

// Schema describe a struct or a property in schema.
// Properties are sort by name, because openapi3 use map to store properties,map is not ordered.
type Schema struct {
	SchemaOptions
	Type        *code.TypeInfo // The type of the schema.
	HasRegular  bool           // if schema has a pattern setting
	validations []string       // the expression string for validator
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

func NewSchema(options ...SchemaOption) *Schema {
	sch := &Schema{
		SchemaOptions: NewSchemaOptions(options...),
	}
	sch.Properties = make(map[string]*Schema)
	return sch
}

func (sch *Schema) AddProperties(name string, schema *Schema) {
	sch.Properties[name] = schema
	sch.properties = append(sch.properties, schema)
}

// GenSchemaType generates the type of the parameter by SPEC.
// schema type from : the schema's sepc , additionalProperties, array items
func (sch *Schema) GenSchemaType(c *Config) {
	var info *code.TypeInfo
	if tm := c.TypeMap[sch.Spec.Ref]; tm != nil {
		sch.Type = tm.Clone()
		return
	}
	var typeName = sch.Name
	if sch.Spec.Ref != "" {
		typeName = schemaNameFromRef(sch.Spec.Ref)
	}
	sv := sch.Spec.Value
	var sepcType string
	if sv.AllOf != nil {
		sepcType = "object"
	} else if len(sv.Type.Slice()) == 0 && (sv.OneOf == nil || sv.AnyOf == nil) {
		// todo check oneOf and anyOf
		sepcType = "object"
	} else if len(sv.Type.Slice()) != 0 {
		sepcType = sv.Type.Slice()[0]
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
		subSch := genSchemaRef(c, sch.With(WithSchemaSpec(sv.Items), WithSchemaName("")))
		sch.ItemSchema = subSch
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
				info = &code.TypeInfo{Type: code.TypeEnum, Ident: helper.Pascal(sch.Name)}
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
				sch.Spec = sv.AdditionalProperties.Schema
				sch.Name = schemaNameFromRef(sv.AdditionalProperties.Schema.Ref)
				sch.GenSchemaType(c)
			} else {
				subSch := genSchemaRef(c, sch.With(WithSchemaSpec(sv.AdditionalProperties.Schema), WithSchemaName("")))
				info.Ident = "map[string]" + subSch.Type.String()
				sch.ItemSchema = subSch
			}
			info.Nillable = true

			break
		} else if isJsonRawObject(c, sch.Name, sv) {
			info.Type = code.TypeJSON
			info.Nillable = true
			break
		}
		info.Ident = helper.Pascal(typeName)
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

	if sch.Spec.Value.Nullable && !info.Nillable {
		if info.Ident != "" {
			info.Ident = "*" + info.Ident
		}
		info.Nillable = true
	}
	sch.Type = info
	return
}

var updateContentTypes = make(map[string]struct{})

// AppendContentTypeStructTag parse content type and append to struct tags, effected on ref schema.
// if depends on Request or Response content type.
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
		if ref != "" {
			if s, ok := c.FindSchema(ref); ok {
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
		}
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
// Most native types will be returned, time.Time will be returned point if not required.
func (sch *Schema) TypeString() string {
	s := sch.Type.String()

	if sch.Required {
		return sch.StructString()
	}
	if sch.Type.Nillable {
		return s
	}
	switch sch.Type.Type {
	case code.TypeInt8, code.TypeInt16, code.TypeInt32, code.TypeInt, code.TypeInt64:
		return s
	case code.TypeUint8, code.TypeUint16, code.TypeUint32, code.TypeUint, code.TypeUint64:
		return s
	case code.TypeFloat32, code.TypeFloat64:
		return s
	case code.TypeString, code.TypeBytes, code.TypeEnum:
		return s
	case code.TypeBool:
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
	if sch.Spec.Value.Type.Is("array") {
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

// BuildFromConfig builds the schema from the config's SchemaMap
func (sch *Schema) BuildFromConfig(c *Config) bool {
	s, ok := c.FindSchema(sch.SchemaOptions.Spec.Ref)

	if !ok {
		return false
	}
	sch.properties = s.properties
	sch.Properties = s.Properties
	sch.StructTags = s.StructTags
	sch.validations = s.validations
	sch.HasRegular = s.HasRegular
	sch.IsReplace = s.IsReplace
	sch.IsInline = s.IsInline
	sch.IsAlias = s.IsAlias
	sch.Type = s.Type
	sch.IsArray = s.IsArray
	sch.ItemSchema = s.ItemSchema
	// if it's an alias, we need to get the name for reference type
	if s.IsAlias {
		tt := *s.Type
		sch.Type = &tt
		sch.Type.Ident = helper.Pascal(s.Name)
	}
	sch.FixRequired()
	return true
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

func (sch *Schema) genItemSchema(c *Config, spec *openapi3.SchemaRef) {
	if spec.Value.Items == nil {
		return
	}
	popt := sch.SchemaOptions.With(WithSchemaSpec(spec.Value.Items), WithSchemaName(sch.Name))
	gs := genSchemaRef(c, popt)
	sch.ItemSchema = gs
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

func (sch *Schema) genProperties(c *Config, spec *openapi3.SchemaRef) {
	for _, pname := range sortPropertyKeys(spec.Value.Properties) {
		schemaRef := sch.Spec.Value.Properties[pname]
		popt := sch.SchemaOptions.With(WithSchemaName(pname),
			WithPrefixName(sch.Name), WithSchemaSpec(schemaRef),
			WithSchemaRequired(helper.InStrSlice(spec.Value.Required, pname)))
		gs := genSchemaRef(c, popt)
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

func (sch *Schema) OrderedProperties() []*Schema {
	return sch.properties
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
	if !schema.Type.Is("object") {
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
