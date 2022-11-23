package codegen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"net/http"
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
			op.GenSecurity(spec.Components.SecuritySchemes)
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
			}
		}
	}
	// response object is pointer
	if !r.Schema.Type.Nillable {
		r.Schema.Type.Nillable = true
		if r.Schema.Type.Ident != "" && !strings.HasPrefix(r.Schema.Type.Ident, "*") {
			r.Schema.Type.Ident = "*" + r.Schema.Type.Ident
		}
	}
	return r
}

func (op *Operation) GenSecurity(ssSpec openapi3.SecuritySchemes) {
	if op.Spec.Security == nil {
		return
	}
	for _, sec := range *op.Spec.Security {
		for name, scopes := range sec {
			ss := ssSpec[name]
			switch ss.Value.Type {
			case "http":
			case "apiKey":
			case "oauth2":
				if len(scopes) == 0 {

				}
			case "openIdConnect":
			default:
				panic(fmt.Errorf("unknown security type:%s", ss.Value.Type))
			}
		}
	}
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
