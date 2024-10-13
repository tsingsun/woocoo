package oasgen

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
)

type SchemaOption func(options *SchemaOptions)

func WithSchemaTag(tag string) SchemaOption {
	return func(options *SchemaOptions) {
		options.Tag = tag
	}
}
func WithSchemaSpec(spec *openapi3.SchemaRef) SchemaOption {
	return func(options *SchemaOptions) {
		options.Spec = spec
		options.IsRef = spec.Ref != ""
	}
}

// WithSchemaName sets the name of the schema.It is from Spec and keep the original name for set go type Tag.
func WithSchemaName(name string) SchemaOption {
	return func(options *SchemaOptions) {
		options.Name = name
	}
}

func WithSchemaRequired(required bool) SchemaOption {
	return func(options *SchemaOptions) {
		options.Required = required
	}
}

func WithSchemaZone(zone SchemaZone) SchemaOption {
	return func(options *SchemaOptions) {
		options.SchemaZone = zone
	}
}

func WithPrefixName(prefix string) SchemaOption {
	return func(options *SchemaOptions) {
		options.PrefixName = helper.Pascal(prefix)
	}
}

// WithNotRef indicates that the schema is not a reference to a component schema.
func WithNotRef() SchemaOption {
	return func(options *SchemaOptions) {
		options.IsRef = false
	}
}

// SchemaOptions helps gen schema
type SchemaOptions struct {
	// Tag is the tag name of the operation.
	Tag string
	// The name of the schema.
	Name string
	// The prefix name of the schema. it can be used in enum type.
	PrefixName string
	// The original OpenAPIv3 Schema.
	Spec *openapi3.SchemaRef
	// Required indicates if the schema is required.
	Required   bool
	SchemaZone SchemaZone
	// IsRef indicates if the schema is a reference to a component schema.If schema is build from `#/components/schema`,
	// value is false.
	IsRef bool
}

func NewSchemaOptions(opts ...SchemaOption) SchemaOptions {
	options := SchemaOptions{}.With(opts...)
	return options
}

// With returns a new SchemaOptions base on source and with the given options.
func (s SchemaOptions) With(opts ...SchemaOption) SchemaOptions {
	ns := s
	for _, opt := range opts {
		opt(&ns)
	}
	//if ns.Spec != nil {
	//	if ns.Spec.Ref != "" {
	//		if strings.HasPrefix(ns.Spec.Ref, ComponentsRefPrefix) {
	//			ns.SchemaZone = SchemaZoneComponent
	//		}
	//	}
	//}
	ns.Named()
	return ns
}

// Named returns the name of the schema, if Name is empty, we will get the name from the ref.
func (s *SchemaOptions) Named() *SchemaOptions {
	if s.Spec.Ref != "" && s.Name == "" { // if it's a ref, we need to get the name from the ref
		s.Name = schemaNameFromRef(s.Spec.Ref)
	}
	return s
}

func (s SchemaOptions) IsComponent() bool {
	return s.SchemaZone == SchemaZoneComponent
}

func (s SchemaOptions) IsResponse() bool {
	return s.SchemaZone == SchemaZoneResponse
}

func (s SchemaOptions) IsRequest() bool {
	return s.SchemaZone == SchemaZoneRequest
}
