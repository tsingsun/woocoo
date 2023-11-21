package oasgen

import "github.com/tsingsun/woocoo/cmd/woco/code"

type Request struct {
	// Name is the name of the request, which is the name of the operation.
	Name string
	// Parameters are the request parameters, including path, query, header, cookie and body.
	Parameters       []*Parameter
	PathParameters   []*Parameter
	QueryParameters  []*Parameter
	HeaderParameters []*Parameter
	CookieParameters []*Parameter
	Body             []*Parameter
	BodyContentTypes []string

	BindKind BindKind
}

func (r *Request) HasPath() bool {
	return len(r.PathParameters) > 0
}

func (r *Request) HasQuery() bool {
	return len(r.QueryParameters) > 0
}

func (r *Request) HasHeader() bool {
	return len(r.HeaderParameters) > 0
}

func (r *Request) HasCookie() bool {
	return len(r.CookieParameters) > 0
}

func (r *Request) HasBody() bool {
	return len(r.Body) > 0
}

func (r *Request) HasMultiBind() bool {
	if r.BindKind == 0 {
		return false
	}
	return r.BindKind != BindKindBody && r.BindKind != BindKindPath &&
		r.BindKind != BindKindQuery && r.BindKind != BindKindHeader && r.BindKind != BindKindCookie
}

func (r *Request) HasDefaultValue() bool {
	for _, p := range r.Parameters {
		if p.Schema.Spec.Value.Default != nil {
			return true
		}
	}
	return false
}

// IndependentSchemas returns the independent schemas of the request parameters which are not in the spec's schema section.
// The top level Schemas are in the parameters. We search the sub schemas recursively, and generate the code to the `tag`
// template.
func (r *Request) IndependentSchemas() []*Schema {
	var schemas []*Schema
	var handler = func(ps []*Parameter) {
		for _, p := range ps {
			schemas = append(schemas, loadIndependentSchema(p.Schema)...)
		}
	}
	handler(r.Parameters)
	return schemas
}

func loadIndependentSchema(sch *Schema) []*Schema {
	if sch.Spec.Ref != "" {
		return nil
	}
	if sch.IsArray && sch.ItemSchema != nil {
		return loadIndependentSchema(sch.ItemSchema)
	}
	var store []*Schema
	switch sch.Type.Type {
	case code.TypeEnum:
		// if native type, no need to generate schema
		if sch.Type.Ident != "" {
			store = append(store, sch)
		}
	case code.TypeOther:
		store = append(store, sch)
	}
	for _, property := range sch.properties {
		switch property.Type.Type {
		case code.TypeOther, code.TypeEnum:
			store = append(store, loadIndependentSchema(property)...)
		}
	}
	return store
}
