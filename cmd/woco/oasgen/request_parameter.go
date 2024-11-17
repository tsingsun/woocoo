package oasgen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tsingsun/woocoo/cmd/woco/code"
)

// Parameter include parameter and requestBody
type Parameter struct {
	Name   string
	Schema *Schema
	Spec   *openapi3.Parameter
	// position index in path uri
	Index int
}

func newParameterFromSchema(c *Config, schema *Schema) *Parameter {
	name := schema.Name
	p := &Parameter{
		Name:   name,
		Schema: schema,
		Spec: &openapi3.Parameter{
			Name:        name,
			In:          "",
			Description: schema.Spec.Value.Description,
			Required:    schema.Required,
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
		if p.Schema.Type.Type == code.TypeTime {

		}
		ts = append(ts, fmt.Sprintf(`form:"%s"`, tagName))
	}
	p.Schema.StructTags = append(p.Schema.StructTags, ts...)
}
