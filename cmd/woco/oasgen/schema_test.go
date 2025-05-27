package oasgen

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"testing"
)

func TestGenComponentSchema_genSchemaRef(t *testing.T) {
	type args struct {
		c    *Config
		opts SchemaOptions
	}
	tests := []struct {
		name string
		args args
		want *Schema
	}{
		{
			name: "map[string]string alias",
			args: args{
				c: &Config{
					Package: "petstore",
				},
				opts: SchemaOptions{
					Name: "labelSet",
					Spec: &openapi3.SchemaRef{
						Ref: "",
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"object"},
							AdditionalProperties: openapi3.AdditionalProperties{
								Schema: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: &openapi3.Types{"string"},
									},
								},
							},
						},
					},
				},
			},
			want: &Schema{
				SchemaOptions: SchemaOptions{
					Name: "labelSet",
				},
				Type: &code.TypeInfo{
					Ident:   "map[string]string",
					PkgName: "petstore",
					Type:    code.TypeOther,
				},
				IsAlias: true,
			},
		},
		{
			name: "enum",
			args: args{
				c: &Config{
					Package: "petstore",
				},
				opts: SchemaOptions{
					Name: "PetStatus",
					Spec: &openapi3.SchemaRef{
						Ref: "",
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
							Enum: []interface{}{
								"available",
								"pending",
								"sold",
							},
						},
					},
				},
			},
			want: &Schema{
				SchemaOptions: SchemaOptions{
					Name: "PetStatus",
				},
				Type: &code.TypeInfo{
					Ident:   "PetStatus",
					PkgName: "petstore",
					Type:    code.TypeEnum,
				},
				IsAlias: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := genSchemaRef(tt.args.c, tt.args.opts)
			got.setAlias()
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.IsAlias, got.IsAlias)
			assert.Equal(t, tt.want.Type.Ident, got.Type.Ident)
		})
	}
}

func Test_genSchemaRef_IncludeAlias(t *testing.T) {
	type args struct {
		c    *Config
		name string
		spec *openapi3.SchemaRef
		opts SchemaOptions
	}
	tests := []struct {
		name string
		args args
		want *Schema
	}{
		{
			name: "alias",
			args: args{
				c: &Config{
					Package: "petstore",
					schemas: map[string]*Schema{
						"#/components/schemas/labelSet": {
							SchemaOptions: SchemaOptions{
								Name: "labelSet",
							},
							Type: &code.TypeInfo{
								Ident: "map[string]string",
							},
							IsAlias: true,
						},
					},
				},
				name: "Tag",
				spec: &openapi3.SchemaRef{
					Ref: "",
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"object"},
						Title:       "Pet Tag",
						Description: "A tag for a pet",
						XML: &openapi3.XML{
							Name: "Tag",
						},
						Properties: map[string]*openapi3.SchemaRef{
							"id": {
								Ref: "",
								Value: &openapi3.Schema{
									Type:   &openapi3.Types{"integer"},
									Format: "int64",
								},
							},
							"labels": {
								Ref: "#/components/schemas/labelSet",
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"object"},
									AdditionalProperties: openapi3.AdditionalProperties{
										Schema: &openapi3.SchemaRef{
											Value: &openapi3.Schema{
												Type: &openapi3.Types{"string"},
											},
										},
									},
								},
							},
							"name": {
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"string"},
								},
							},
						},
					},
				},
			},
			want: &Schema{
				SchemaOptions: SchemaOptions{
					Name: "Tag",
				},
				Type: &code.TypeInfo{
					Ident:   "*Tag",
					PkgName: "petstore",
				},
				Properties: map[string]*Schema{
					"id": {
						SchemaOptions: SchemaOptions{
							Name: "id",
						},
						Type: &code.TypeInfo{
							Type: code.TypeInt64,
						},
					},
					"labels": {
						SchemaOptions: SchemaOptions{
							Name:  "labels",
							IsRef: true,
						},
						Type: &code.TypeInfo{
							Ident:    "LabelSet",
							Type:     code.TypeOther,
							Nillable: true,
						},
					},
					"name": {
						SchemaOptions: SchemaOptions{
							Name: "name",
						},
						Type: &code.TypeInfo{
							Type: code.TypeString,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.opts.Spec = tt.args.spec
			tt.args.opts.Name = tt.args.name
			got := genSchemaRef(tt.args.c, tt.args.opts)
			assert.Equal(t, tt.want.Properties["labels"].Name, got.Properties["labels"].Name)
			assert.Equal(t, tt.want.Properties["labels"].IsRef, got.Properties["labels"].IsRef)
			assert.Equal(t, tt.want.Properties["labels"].Type.Ident, got.Properties["labels"].Type.Ident)
		})
	}
}
