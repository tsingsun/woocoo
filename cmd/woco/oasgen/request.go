package oasgen

type Request struct {
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
