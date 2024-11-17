package oasgen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"strconv"
)

// genComponentSchemas generate component schemas
//
// make sure this method is run first,because type map will change in genSchemaRef by request or response
// and isReplace will be not correct
func genComponentSchemas(c *Config, spec *openapi3.T) {
	tmpTypeMap := make(map[string]*code.TypeInfo)
	for k, v := range c.TypeMap {
		tmpTypeMap[k] = v
	}
	for _, name := range sortPropertyKeys(spec.Components.Schemas) {
		schemaRef := spec.Components.Schemas[name]
		k := ComponentsRefPrefix + name
		gs := genSchemaRef(c, NewSchemaOptions(WithSchemaName(name), WithSchemaSpec(schemaRef), WithSchemaZone(SchemaZoneComponent)))
		if tp, ok := tmpTypeMap[k]; ok { // schemas do not has a ref
			gs.IsReplace = true
			gs.Type = tp
		}
		c.AddTypeMap(k, gs.Type)
		c.AddSchema(name, gs)
	}
	return
}

// generate parameter from spec. the spec must have schema or content
// TODO Content Now only support application/json
func genParameter(c *Config, op *Operation, spec *openapi3.ParameterRef) *Parameter {
	pv := spec.Value
	name := spec.Value.Name
	ep := &Parameter{
		Name: name,
		Spec: pv,
	}
	switch {
	case pv.Schema != nil:
		ep.Schema = genSchemaRef(c, SchemaOptions{
			PrefixName: op.Name,
			Name:       name,
			SchemaZone: SchemaZoneRequest,
			Spec:       pv.Schema,
			Required:   ep.Spec.Required})
		if !ep.Spec.Required {
			ep.Schema.Type.AsPointer()
		}
	case pv.Content != nil:
		mt, ok := pv.Content["application/json"]
		if !ok {
			return ep
		}
		ep.Schema = genSchemaRef(c, SchemaOptions{
			Name:       name,
			SchemaZone: SchemaZoneRequest,
			Spec:       mt.Schema,
		})
	default:
		panic(fmt.Errorf("parameter %s must have Spec or content", pv.Name))
	}
	ep.initStructTag()
	return ep
}

// genSchemaRef generate schema and store the schema into cache by options.
//
// Schema is skip to cache if option.SkipAdd is true or is go native type.
func genSchemaRef(c *Config, option SchemaOptions) (sch *Schema) {
	sch = &Schema{
		SchemaOptions: option,
		Properties:    make(map[string]*Schema),
	}
	defer func() {
		switch {
		case sch.IsArray:
			return
		case sch.Name == "":
			// if native type, no need to generate schema
			return
		case sch.Type.Ident == "":
			// if native type, no need to generate schema
			return
		case sch.SkipAdd:
			return
		}
		switch sch.Type.Type {
		case code.TypeOther:
			c.AddSchema(sch.Name, sch)
		case code.TypeEnum:
			key := helper.Pascal(sch.PrefixName) + helper.Pascal(sch.Name)
			c.AddSchema(key, sch)
		}
	}()
	if sch.IsRef {
		// generate dependent schema
		_, ok := c.FindSchema(sch.Spec.Ref)
		if !ok {
			genSchemaRef(c, option.With(WithComponent(option.Spec)))
		}
		ok = sch.BuildFromConfig(c)
		if ok {
			return sch
		}
	}
	sch.GenSchemaType(c)
	sch.FixRequired()
	for k, v := range sch.Spec.Value.Extensions {
		switch k {
		case goTag:
			sch.StructTags = append(sch.StructTags, v.(string))
		}
	}

	// allOf, make sub schema as properties
	if sch.Spec.Value != nil && len(sch.Spec.Value.AllOf) > 0 {
		// allof node is a new schema
		sch.IsRef = false
		for i, one := range sch.Spec.Value.AllOf {
			if one.Ref != "" {
				gs := genSchemaRef(c, option.With(WithSchemaSpec(one), WithSchemaName("")))
				// inline struct
				gs.IsInline = true
				sch.AddProperties(strconv.Itoa(i), gs)
			} else {
				for _, oname := range sortPropertyKeys(one.Value.Properties) {
					schemaRef := one.Value.Properties[oname]
					gs := genSchemaRef(c, option.With(WithSchemaSpec(schemaRef), WithSchemaName(oname),
						WithSchemaRequired(helper.InStrSlice(one.Value.Required, oname))))
					sch.AddProperties(oname, gs)
				}
			}
		}
	}
	sch.CollectTags()
	sch.setAlias()
	// set to c.schemas , and avoid recursive
	if sch.IsRef {
		if sch.IsAlias {
			// not the root schema, set alias to type todo set type other field
			sch.Type.Ident = helper.Pascal(schemaNameFromRef(sch.Spec.Ref))
		}
	}
	// if component schema is enum, we need to set the ident of the type to avoid conflict
	if sch.IsEnum() && sch.IsComponent() {
		if sch.ItemSchema != nil {
			sch.ItemSchema.Type.Ident = helper.Pascal(sch.PrefixName) + helper.Pascal(sch.Name)
		} else {
			sch.Type.Ident = helper.Pascal(sch.PrefixName) + helper.Pascal(sch.Name)
		}
	}
	sch.genProperties(c, sch.Spec)
	return sch
}
