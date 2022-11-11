package codegen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"net/http"
	"reflect"
	"sort"
	"strconv"
)

type (
	// Operation is for operation of openapi3
	Operation struct {
		*Config
		Name         string
		Group        string // first tag name
		Method       string // GET, POST, DELETE, etc.
		Path         string
		Spec         *openapi3.Operation
		SpecPathItem *openapi3.PathItem // navigation to pathItem
		Request      *Request
		Responses    []*Response
		ResponseOK   *Response
	}
	// Parameter include parameter and requestBody
	Parameter struct {
		Name   string
		Schema *Schema
		Spec   *openapi3.Parameter
	}
	Request struct {
		Parameters []*Parameter
		BindUri    bool
		BindHeader bool
		BindCookie bool
		Bind       bool
	}
	// Response is for response of openapi3
	Response struct {
		Name string
		// http status
		Status int
		Schema *Schema
		Spec   *openapi3.Response
	}
	// Schema is for schema of openapi3
	Schema struct {
		Spec       *openapi3.SchemaRef // The original OpenAPIv3 Schema.
		Name       string
		Type       *code.TypeInfo
		IsRef      bool
		Required   bool
		StructTags []string
		properties []*Schema
		Properties map[string]*Schema
	}
	// Tag is for tag of openapi3
	Tag struct {
		*Config
		Name       string
		Operations []*Operation
		Spec       *openapi3.Tag
	}
)

func genOperation(c *Config, schema *openapi3.T) (ops []*Operation) {
	// sort Spec.Paths by path
	keys := make([]string, 0, len(schema.Paths))
	for k := range schema.Paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		pathItem := schema.Paths[key]
		opmap := pathItem.Operations()
		for _, method := range sortSpecOperationKeys(opmap) {
			specop := opmap[method]
			tag := ""
			if len(specop.Tags) > 0 {
				tag = specop.Tags[0]
			}
			op := &Operation{
				Name:         helper.Pascal(specop.OperationID),
				Method:       method,
				Path:         key,
				Spec:         specop,
				SpecPathItem: pathItem,
				Group:        tag,
				Request:      &Request{},
			}
			ops = append(ops, op)
			for _, p := range op.Spec.Parameters {
				gp := genParameter(c, p)
				op.AddParameter(gp)
				switch gp.Spec.In {
				case "path":
					op.Request.BindUri = true
				case "header":
					op.Request.BindHeader = true
				case "cookie":
					op.Request.BindCookie = true
				case "query", "form":
					op.Request.Bind = true
				}
			}
			if rb := op.Spec.RequestBody; rb != nil {
				op.AddParameter(genParameterFromContent(c, op.RequestName(), rb.Value.Content)...)
				op.Request.Bind = true
			}

			if rs := op.Spec.Responses; rs != nil {
				for _, name := range sortSpecResponseKeys(rs) {
					status := name
					if name == "default" {
						status = "0"
					}
					res := genResponse(c, status, rs[name])
					op.Responses = append(op.Responses, res)
					if res.Status == http.StatusOK {
						op.ResponseOK = res
					}
				}
			}
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Name < ops[j].Name
	})
	return
}

func genParameter(c *Config, p *openapi3.ParameterRef) *Parameter {
	pv := p.Value
	name := p.Value.Name
	ep := &Parameter{
		Name: name,
		Spec: pv,
	}
	switch {
	case pv.Schema != nil:
		ep.Schema = genSchemaRef(c, name, pv.Schema)
	case pv.Content != nil:
		mt, ok := pv.Content["application/json"]
		if !ok {
			return ep
		}
		ep.Schema = genSchemaRef(c, name, mt.Schema)
	default:
		panic(fmt.Errorf("parameter %s must have Spec or content", pv.Name))
	}
	ep.initStructTag()
	return ep
}

func genParameterFromContent(c *Config, name string, content openapi3.Content) (params []*Parameter) {
	for _, mediaType := range content {
		ps := genSchemaRef(c, name, mediaType.Schema)
		if mediaType.Schema.Ref == "" { // from independent Spec
			for _, property := range ps.properties {
				param := newParameterFromSchema(c, property)
				param.initStructTag()
				params = append(params, param)
			}
			return
		}
		// from reference Spec
		param := newParameterFromSchema(c, ps)
		param.initStructTag()
		params = append(params, param)
		break
	}
	return
}

func genResponse(c *Config, codeStr string, response *openapi3.ResponseRef) *Response {
	if response == nil {
		return nil
	}
	status, err := strconv.Atoi(codeStr)
	if err != nil {
		panic(fmt.Errorf("response status code must be int:%s", codeStr))
	}
	if response.Value.Content == nil {
		return &Response{
			Status: status,
		}
	}
	r := &Response{
		Status: status,
		Spec:   response.Value,
	}
	// use first content type
	for _, mediaType := range response.Value.Content {
		r.Schema = genSchemaRef(c, "", mediaType.Schema)
		break
	}
	if status == http.StatusOK {
		switch r.Schema.Type.Type {
		case code.TypeOther:
			if r.Schema.Type.Ident != "" && !r.Schema.Type.Nillable {
				r.Schema.Type.Ident = "*" + r.Schema.Type.Ident
			}
		}
	}
	if r.Name == "" {
		r.Name = "data"
	}
	return r
}

func genSchemaRef(c *Config, name string, schema *openapi3.SchemaRef) *Schema {
	sv := schema.Value
	sc := &Schema{
		Name:       name,
		Spec:       schema,
		Properties: make(map[string]*Schema),
	}
	st, err := schemaToType(c, name, schema)
	if err != nil {
		panic(err)
	}
	sc.Type = st
	sc.IsRef = schema.Ref != ""
	if sc.IsRef {
		sc.Name = schemaNameFromRef(schema.Ref)
	}
	if sc.Type.Type == code.TypeOther {
		// inline
		if sc.Name == "" {
			switch sc.Spec.Value.Type {
			case "array":
				sc.Name = code.TypeName(sc.Type.Ident) + "List"
				sc.IsRef = schema.Value.Items.Ref != ""
			}
		}
	}

	for _, name := range sortPropertyKeys(sv.Properties) {
		schemaRef := sv.Properties[name]
		gs := genSchemaRef(c, name, schemaRef)
		gs.Name = name
		gs.Required = helper.InStrSlice(schema.Value.Required, name)
		if !gs.Required {
			if gs.Type.Type == code.TypeOther && !gs.Type.Nillable {
				gs.Type.Ident = "*" + gs.Type.Ident
			}
		}
		jsTag := fmt.Sprintf(`json:"%s"`, gs.Name)
		if !gs.Required {
			jsTag = fmt.Sprintf(`json:"%s,omitempty"`, gs.Name)
		}
		gs.StructTags = append(gs.StructTags, jsTag)

		sc.Properties[name] = gs
		sc.properties = append(sc.properties, gs)
	}
	return sc
}

func genComponentSchemas(c *Config, schema *openapi3.T) (schemas []*Schema) {
	for _, name := range sortPropertyKeys(schema.Components.Schemas) {
		schemaRef := schema.Components.Schemas[name]
		gs := genSchemaRef(c, name, schemaRef)
		schemas = append(schemas, gs)
	}
	return
}

func schemaToType(c *Config, name string, schema *openapi3.SchemaRef) (info *code.TypeInfo, err error) {
	if schema == nil {
		return
	}
	if tm := c.TypeMap[schema.Ref]; tm != nil {
		info = tm.Clone()
		return info, nil
	}
	sv := schema.Value
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
		item, err := schemaToType(c, itemName, sv.Items)
		if err != nil {
			return nil, err
		}
		if item.RType == nil {
			info.Ident = "[]" + item.String()
			info.Type = code.TypeOther
		} else {
			iv := reflect.MakeSlice(item.RType.ReflectType(), 0, 0)
			rt, err := code.ParseGoType(iv)
			if err != nil {
				return nil, err
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
		case "date-time":
			info = &code.TypeInfo{Type: code.TypeTime, PkgPath: "time"}
		case "uuid":
			rt, err := code.ParseGoType(uuid.New())
			if err != nil {
				return nil, err
			}
			info = &code.TypeInfo{Type: code.TypeUUID, RType: rt, Ident: rt.Ident, PkgPath: rt.PkgPath, PkgName: code.PkgName(rt.String())}
		case "json":
			info = &code.TypeInfo{Type: code.TypeJSON, Nillable: true}
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
		ref := schema.Ref
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
		err = fmt.Errorf("unhandled OpenAPISchema type: %s", sv.Type)
	}
	if info == nil {
		return
	}
	if schema.Value.Nullable && !info.Nillable {
		info.Ident = "*" + info.Ident
		info.Nillable = true
	}
	return
}

func (o *Operation) AddParameter(params ...*Parameter) {
	o.Request.Parameters = append(o.Request.Parameters, params...)
}

func (o Operation) HasRequest() bool {
	return len(o.Request.Parameters) > 0
}

func (o Operation) HasResponse() bool {
	return o.ResponseOK != nil
}

// RequestName returns the name of the request struct.
func (o Operation) RequestName() string {
	return o.Name + "Request"
}

func (o *Tag) PackageDir() string {
	if o.Name == defaultTagName {
		return ""
	}
	return o.Name
}

// AddOperation adds an operation to the Tag.
func (t *Tag) AddOperation(ops ...*Operation) {
	t.Operations = append(t.Operations, ops...)
}

// InterfaceName for Tag
func (t *Tag) InterfaceName() string {
	if t.Name == defaultTagName {
		return "Server"
	}
	return t.Name
}

func newParameterFromSchema(c *Config, schema *Schema) *Parameter {
	name := schema.Name
	p := &Parameter{
		Name:   name,
		Schema: schema,
		Spec: &openapi3.Parameter{
			Name: name,
			In:   "",
		},
	}
	return p
}

func (p *Parameter) initStructTag() {
	tagName := p.Name
	ts := make([]string, 0, 2)
	switch p.Spec.In {
	case "path":
		// {:id}
		ts = append(ts, fmt.Sprintf(`uri:"%s"`, tagName))
	case "header":
		ts = append(ts, fmt.Sprintf(`header:"%s"`, tagName))
	case "cookie":
		ts = append(ts, fmt.Sprintf(`cookie:"%s"`, tagName))
	case "query":
		fallthrough
	default:
		// query /id/ or form , body
		ts = append(ts, fmt.Sprintf(`form:"%s"`, tagName))
	}
	p.Schema.Required = p.Spec.Required
	if p.Schema.Required {
		ts = append(ts, `binding:"required"`)
	}
	p.Schema.StructTags = ts
}

func sortSpecOperationKeys(spec map[string]*openapi3.Operation) []string {
	keys := make([]string, 0, len(spec))
	for k := range spec {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortPropertyKeys(spec openapi3.Schemas) []string {
	keys := make([]string, 0, len(spec))
	for k := range spec {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortSpecResponseKeys(rs openapi3.Responses) []string {
	keys := make([]string, 0, len(rs))
	for k := range rs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
