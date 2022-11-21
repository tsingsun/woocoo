package codegen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const (
	TagRegular = "regex"
)

type (
	// Operation is for operation of openapi3
	Operation struct {
		*Config
		Name             string
		Group            string // first tag name
		Method           string // GET, POST, DELETE, etc.
		Path             string
		Spec             *openapi3.Operation
		SpecPathItem     *openapi3.PathItem // navigation to pathItem
		Request          *Request
		Responses        []*Response
		ResponseOK       *Response
		ResponseNotFound *Response
	}
	// Parameter include parameter and requestBody
	Parameter struct {
		Name   string
		Schema *Schema
		Spec   *openapi3.Parameter
	}
	Request struct {
		Parameters       []*Parameter
		BindUri          bool
		UriParameters    []*Parameter
		BindHeader       bool
		HeaderParameters []*Parameter
		BindCookie       bool
		CookieParameters []*Parameter
		BindBody         bool
		Body             []*Parameter
		BodyContentTypes []string
	}
	// Response is for response of openapi3
	Response struct {
		Name string
		// return content type when response is not empty
		ContentTypes []string
		// http status
		Status      int
		Schema      *Schema
		Spec        *openapi3.Response
		Description *string
	}
	// Schema is for schema of openapi3
	Schema struct {
		Spec       *openapi3.SchemaRef // The original OpenAPIv3 Schema.
		Name       string
		Type       *code.TypeInfo
		IsRef      bool
		HasRegular bool // if schema has a pattern setting
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

func genOperation(c *Config, spec *openapi3.T) (ops []*Operation) {
	// sort Spec.Paths by path
	keys := make([]string, 0, len(spec.Paths))
	for k := range spec.Paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		pathItem := spec.Paths[key]
		opmap := pathItem.Operations()
		for _, method := range sortSpecOperationKeys(opmap) {
			specop := opmap[method]
			tag := ""
			if len(specop.Tags) > 0 {
				tag = specop.Tags[0]
			}
			op := &Operation{
				Config:       c,
				Name:         helper.Pascal(specop.OperationID),
				Method:       method,
				Path:         key,
				Spec:         specop,
				SpecPathItem: pathItem,
				Group:        tag,
				Request:      &Request{},
			}
			ops = append(ops, op)
			op.GenParameters()
			op.GenResponses()
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Name < ops[j].Name
	})
	return
}

func genParameter(c *Config, spec *openapi3.ParameterRef) *Parameter {
	pv := spec.Value
	name := spec.Value.Name
	ep := &Parameter{
		Name: name,
		Spec: pv,
	}
	switch {
	case pv.Schema != nil:
		ep.Schema = genSchemaRef(c, name, pv.Schema, ep.Spec.Required)
	case pv.Content != nil:
		mt, ok := pv.Content["application/json"]
		if !ok {
			return ep
		}
		ep.Schema = genSchemaRef(c, name, mt.Schema, false)
	default:
		panic(fmt.Errorf("parameter %s must have Spec or content", pv.Name))
	}
	ep.initStructTag()
	return ep
}

func genParameterFromContent(c *Config, name string, content openapi3.Content, required bool) (params []*Parameter) {
	var schema *Schema
	contentTypes := make([]string, 0, len(content))
	for ct, mediaType := range content {
		contentTypes = append(contentTypes, ct)
		if schema == nil {
			schema = genSchemaRef(c, name, mediaType.Schema, required)
			if mediaType.Schema.Ref == "" { // from independent Spec
				for _, property := range schema.properties {
					param := newParameterFromSchema(c, property)
					params = append(params, param)
				}
			} else {
				// from reference Spec
				param := newParameterFromSchema(c, schema)
				params = append(params, param)
			}
		}
	}
	for _, param := range params {
		param.initStructTag()
		for _, contentType := range contentTypes {
			param.Schema.AppendContentTypeStructTag(c, param.Name, contentType)
		}
	}
	return
}

func genResponse(c *Config, codeStr string, spec *openapi3.ResponseRef) *Response {
	if spec == nil {
		return nil
	}
	status, err := strconv.Atoi(codeStr)
	if err != nil {
		panic(fmt.Errorf("response status code must be int:%s", codeStr))
	}
	r := &Response{
		Status:      status,
		Spec:        spec.Value,
		Description: spec.Value.Description,
	}
	if spec.Value.Content == nil {
		return r
	}
	// use first content type
	for _, name := range sortSpecMediaTypeKeys(spec.Value.Content) {
		mediaType := spec.Value.Content[name]
		r.ContentTypes = append(r.ContentTypes, name)
		if r.Schema == nil {
			r.Schema = genSchemaRef(c, "", mediaType.Schema, false)
		}
	}
	if status == http.StatusOK {
		if v, ok := c.schemas[r.Schema.Spec.Ref]; ok {
			for _, contentType := range r.ContentTypes {
				v.AppendContentTypeStructTag(c, v.Name, contentType)
				//for _, prop := range v.Properties {
				//	prop.AppendContentTypeStructTag(c,prop.Name, contentType)
				//}
			}
		}
	}
	// all response object is pointer
	r.Schema.Type.Nillable = true
	return r
}

func genSchemaRef(c *Config, name string, spec *openapi3.SchemaRef, required bool) *Schema {
	sv := spec.Value
	sc := &Schema{
		Name:       name,
		Spec:       spec,
		Properties: make(map[string]*Schema),
		Required:   required,
	}
	st, err := schemaToType(c, name, spec)
	if err != nil {
		panic(err)
	}
	sc.Type = st
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
		schemas[k] = gs
	}
	return
}

func schemaToType(c *Config, name string, spec *openapi3.SchemaRef) (info *code.TypeInfo, err error) {
	if spec == nil {
		return
	}
	if tm := c.TypeMap[spec.Ref]; tm != nil {
		info = tm.Clone()
		return info, nil
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
		err = fmt.Errorf("unhandled OpenAPISchema type: %s", sv.Type)
	}
	if info == nil {
		return
	}
	if spec.Value.Nullable && !info.Nillable {
		info.Ident = "*" + info.Ident
		info.Nillable = true
	}
	return
}

func (op *Operation) GenParameters() {
	for _, p := range op.Spec.Parameters {
		gp := genParameter(op.Config, p)
		op.AddParameter(gp)
		switch gp.Spec.In {
		case "path":
			op.Request.BindUri = true
			op.Request.UriParameters = append(op.Request.UriParameters, gp)
		case "header":
			op.Request.BindHeader = true
			op.Request.HeaderParameters = append(op.Request.HeaderParameters, gp)
		case "cookie":
			op.Request.BindCookie = true
			op.Request.CookieParameters = append(op.Request.CookieParameters, gp)
		case "query", "form":
			op.Request.BindBody = true
			op.Request.Body = append(op.Request.Body, gp)
		}
	}
	if rb := op.Spec.RequestBody; rb != nil {
		gps := genParameterFromContent(op.Config, op.RequestName(), rb.Value.Content, rb.Value.Required)
		op.AddParameter(gps...)
		op.Request.BindBody = true
		op.Request.Body = append(op.Request.Body, gps...)
	}
}

func (op *Operation) GenResponses() {
	if rs := op.Spec.Responses; rs != nil {
		for _, name := range sortSpecResponseKeys(rs) {
			status := name
			if name == "default" {
				status = "0"
			}
			res := genResponse(op.Config, status, rs[name])
			op.Responses = append(op.Responses, res)
			switch res.Status {
			case http.StatusOK:
				op.ResponseOK = res
			case http.StatusNotFound:
				op.ResponseNotFound = res
			}
		}
	}
}

func (op *Operation) AddParameter(params ...*Parameter) {
	op.Request.Parameters = append(op.Request.Parameters, params...)
}

func (op *Operation) HasRequest() bool {
	return len(op.Request.Parameters) > 0
}

func (op *Operation) HasResponse() bool {
	return op.ResponseOK != nil
}

// RequestName returns the name of the request struct.
func (op *Operation) RequestName() string {
	return op.Name + "Request"
}

func (t *Tag) PackageDir() string {
	if t.Name == defaultTagName {
		return ""
	}
	return t.Name
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
	case "":
		// from content
		break
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
	p.Schema.StructTags = append(p.Schema.StructTags, ts...)
}

func (sch *Schema) GenSchemaType() {

}

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

func (sch *Schema) CollectTags() {
	var bdex []string
	if sch.Required {
		bdex = append(bdex, "required")
	}
	specValue := sch.Spec.Value
	if specValue.Pattern != "" {
		sch.HasRegular = true
		rName := AddPattern(specValue.Pattern)
		pattenMap[specValue.Pattern] = rName
		tn := fmt.Sprintf("%s=%s", TagRegular, rName)
		bdex = append(bdex, tn)
	}
	if sch.Type.Numeric() {
		if specValue.Max != nil {
			op := "lte"
			if specValue.ExclusiveMax {
				op = "lt"
			}
			bdex = append(bdex, fmt.Sprintf("%s=%v", op, *specValue.Max))
		}
		if specValue.Min != nil {
			op := "gte"
			if specValue.ExclusiveMin {
				op = "gt"
			}
			bdex = append(bdex, fmt.Sprintf("%s=%v", op, *specValue.Min))
		}
	}
	if sch.Type.Stringer() {
		if specValue.MaxLength != nil {
			bdex = append(bdex, fmt.Sprintf("max=%d", *specValue.MaxLength))
		}
		if specValue.MinLength != 0 {
			bdex = append(bdex, fmt.Sprintf("min=%d", specValue.MinLength))
		}
	}
	if sch.Spec.Value.Type == "array" {
		if specValue.MaxItems != nil {
			bdex = append(bdex, fmt.Sprintf("max=%d", specValue.MaxItems))
		}
		if specValue.MinItems != 0 {
			bdex = append(bdex, fmt.Sprintf("min=%d", specValue.MinItems))
		}
		if specValue.UniqueItems {
			bdex = append(bdex, "unique")
		}
	}

	if len(bdex) > 0 {
		sch.StructTags = append(sch.StructTags, fmt.Sprintf(`binding:"%s"`, strings.Join(bdex, ",")))
	}
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

func sortSpecMediaTypeKeys(ps openapi3.Content) []string {
	keys := make([]string, 0, len(ps))
	for k := range ps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
