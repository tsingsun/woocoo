package codegen

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

// Schema is for schema of openapi3
type Schema struct {
	Spec        *openapi3.SchemaRef // The original OpenAPIv3 Schema.
	Name        string
	Type        *code.TypeInfo
	IsRef       bool
	HasRegular  bool // if schema has a pattern setting
	Required    bool
	validations []string // the expression string for validator
	StructTags  []string
	properties  []*Schema
	Properties  map[string]*Schema
	IsReplace   bool // if schema is replaced by model defined in config
}

// GenSchemaType generates the type of the parameter by SPEC.
func (sch *Schema) GenSchemaType(c *Config, name string, spec *openapi3.SchemaRef) {
	var info *code.TypeInfo
	if spec == nil {
		spec = sch.Spec
	}
	if tm := c.TypeMap[spec.Ref]; tm != nil {
		sch.Type = tm.Clone()
		return
	}
	sv := spec.Value
	switch sv.Type {
	case "array":
		itemName := ""
		info = &code.TypeInfo{
			Type:     code.TypeOther,
			Nillable: true,
		}
		if sv.Items.Ref != "" {
			itemName = schemaNameFromRef(sv.Items.Ref)
		}
		sch.GenSchemaType(c, itemName, sv.Items)
		if sch.Type.RType == nil {
			if sch.Type.Type == code.TypeOther {
				info.Ident = "[]*" + sch.Type.String()
			} else {
				info.Ident = "[]" + sch.Type.String()
			}
			info.Type = code.TypeOther
		} else {
			iv := reflect.MakeSlice(sch.Type.RType.ReflectType(), 0, 0)
			rt, err := code.ParseGoType(iv)
			if err != nil {
				panic(err)
			}
			info.Ident = rt.String()
			info.PkgPath = rt.PkgPath
			info.RType = rt
		}
	case "integer":
		switch sv.Format {
		case "int64":
			info = &code.TypeInfo{Type: code.TypeInt64}
		case "int32":
			info = &code.TypeInfo{Type: code.TypeInt32}
		case "int16":
			info = &code.TypeInfo{Type: code.TypeInt16}
		case "int8":
			info = &code.TypeInfo{Type: code.TypeInt8}
		case "uint64":
			info = &code.TypeInfo{Type: code.TypeUint64}
		case "uint32":
			info = &code.TypeInfo{Type: code.TypeUint32}
		case "uint16":
			info = &code.TypeInfo{Type: code.TypeUint16}
		case "uint8":
			info = &code.TypeInfo{Type: code.TypeUint8}
		default:
			info = &code.TypeInfo{Type: code.TypeInt}
		}
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
			sch.validations = append(sch.validations, `datetime=2006-01-02`)
		case "date-time":
			info = &code.TypeInfo{Type: code.TypeTime, PkgPath: "time"}
			sch.validations = append(sch.validations, fmt.Sprintf("datetime=%s", time.RFC3339))
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
			sch.StructTags = append(sch.StructTags, "hostname_rfc1123")
		case "ip":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.StructTags = append(sch.StructTags, "ip")
		case "ipv4":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.StructTags = append(sch.StructTags, "ipv4")
		case "ipv6":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.StructTags = append(sch.StructTags, "ipv6")
		case "uri":
			info = &code.TypeInfo{Type: code.TypeString}
			sch.StructTags = append(sch.StructTags, "uri")
		default:
			info = &code.TypeInfo{Type: code.TypeString}
		}
	case "binary":
		info = &code.TypeInfo{Type: code.TypeBytes, Nillable: true}
	case "object":
		info = &code.TypeInfo{Type: code.TypeOther}
		if sv.AdditionalProperties != nil {
			info.Ident = "any"
			info.Nillable = true
			break
		}
		ref := spec.Ref
		// inline object,generate in package path
		if ref != "" {
			info.Ident = schemaNameFromRef(ref)
			c.AddTypeMap(ref, info)
		} else {
			info.Ident = helper.Pascal(name)
		}
		info.PkgPath = c.Package
		info.PkgName = code.PkgShortName(c.Package)
	default:
		panic(fmt.Errorf("unhandled OpenAPISchema type: %s", sv.Type))
	}
	if info == nil {
		return
	}
	if spec.Value.Nullable && !info.Nillable {
		info.Ident = "*" + info.Ident
		info.Nillable = true
	}
	sch.Type = info
	return
}

// AppendContentTypeStructTag parse content type and append to struct tags
func (sch *Schema) AppendContentTypeStructTag(c *Config, tagName, contentType string) {
	switch contentType {
	case binding.MIMEJSON:
		if HasTag(sch.StructTags, "json") {
			break
		}
		if sch.Required {
			sch.StructTags = append(sch.StructTags, fmt.Sprintf(`json:"%s"`, tagName))
		} else {
			sch.StructTags = append(sch.StructTags, fmt.Sprintf(`json:"%s,omitempty"`, tagName))
		}
	case binding.MIMEPOSTForm:
		if HasTag(sch.StructTags, "form") {
			break
		}
		sch.StructTags = append(sch.StructTags, fmt.Sprintf(`form:"%s"`, tagName))
	case binding.MIMEXML, binding.MIMEXML2:
		if HasTag(sch.StructTags, "xml") {
			break
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
			break
		}
		sch.StructTags = append(sch.StructTags, fmt.Sprintf(`msgpack:"%s"`, tagName))
	default:
		if HasTag(sch.StructTags, "form") {
			break
		}
		sch.StructTags = append(sch.StructTags, fmt.Sprintf(`form:"%s"`, tagName))
	}

	if sch.IsRef {
		ref := sch.Spec.Ref
		if sch.IsObjectArray() {
			// array
			ref = sch.Spec.Value.Items.Ref
		}
		if s, ok := c.schemas[ref]; ok {
			s.AppendContentTypeStructTag(c, s.Name, contentType)
		}
	}
	for _, property := range sch.properties {
		property.AppendContentTypeStructTag(c, property.Name, contentType)
	}
}

// IsObjectArray returns true if the schema is an array of objects
func (sch *Schema) IsObjectArray() bool {
	if sch.Type.Type == code.TypeOther {
		// inline
		switch sch.Spec.Value.Type {
		case "array":
			return true
		}
	}
	return false
}

// CollectTags collects all struct tags from the schema and its children.
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
			s, err := extString(v)
			if err != nil {
				panic(err)
			}
			sch.validations = append(sch.validations, s)
		}
	}
	if sch.Required {
		sch.validations = append(sch.validations, "required")
	} else {
		// if not required, add omitempty as needed
		if len(sch.validations) > 0 {
			sch.validations = append(sch.validations, "omitempty")
		}
	}
	if len(sch.validations) > 0 {
		sch.StructTags = append(sch.StructTags, fmt.Sprintf(`binding:"%s"`, strings.Join(sch.validations, ",")))
	}
}

func genSchemaRef(c *Config, name string, spec *openapi3.SchemaRef, required bool) *Schema {
	sv := spec.Value
	sc := &Schema{
		Name:       name,
		Spec:       spec,
		Properties: make(map[string]*Schema),
		Required:   required,
	}
	sc.GenSchemaType(c, name, spec)
	sc.IsRef = spec.Ref != ""
	if sc.IsRef {
		sc.Name = helper.Camel(schemaNameFromRef(spec.Ref))
	}
	if sc.IsObjectArray() {
		if sc.Name == "" {
			sc.Name = code.TypeName(sc.Type.Ident) + "List"
		}
		sc.IsRef = spec.Value.Items.Ref != ""
	}

	if !sc.Required {
		if sc.Type.Type == code.TypeOther && !sc.Type.Nillable {
			sc.Type.Ident = "*" + sc.Type.Ident
		}
	}
	for k, v := range spec.Value.Extensions {
		switch k {
		case goTag:
			s, err := extString(v)
			if err != nil {
				panic(err)
			}
			sc.StructTags = append(sc.StructTags, s)
		}
	}
	sc.CollectTags()
	for _, name := range sortPropertyKeys(sv.Properties) {
		schemaRef := sv.Properties[name]
		gs := genSchemaRef(c, name, schemaRef, helper.InStrSlice(spec.Value.Required, name))
		sc.Properties[name] = gs
		sc.properties = append(sc.properties, gs)
	}
	return sc
}

func genComponentSchemas(c *Config, spec *openapi3.T) (schemas map[string]*Schema) {
	schemas = make(map[string]*Schema)
	for _, name := range sortPropertyKeys(spec.Components.Schemas) {
		schemaRef := spec.Components.Schemas[name]
		k := "#/components/schemas/" + name
		gs := genSchemaRef(c, name, schemaRef, false)
		if _, ok := c.Models[k]; ok {
			gs.IsReplace = true
		}
		schemas[k] = gs
	}
	return
}
