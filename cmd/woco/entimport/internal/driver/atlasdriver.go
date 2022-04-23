package driver

import (
	"ariga.io/atlas/sql/schema"
	"context"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/entc/load"
	"errors"
	"fmt"
)

type GqlOrderField struct {
	name string
}

func (g GqlOrderField) Name() string {
	return g.name
}

type AtlasBase struct {
	schema.Inspector

	Options ImportOptions
	Types   map[string]*gen.Type

	resolverDsn   func(dsn string) (db string, err error)
	resolverField func(column *schema.Column) (f *load.Field, err error)
}

func (ab *AtlasBase) SchemaInspect(ctx context.Context) ([]*gen.Type, error) {
	inspectOptions := &schema.InspectOptions{
		Tables: ab.Options.Tables,
	}
	// dsn example: root:pass@tcp(localhost:3308)/test?parseTime=True
	dbName, err := ab.resolverDsn(ab.Options.dsn)
	if err != nil {
		return nil, err
	}
	if dbName == "" {
		return nil, errors.New("DSN connection string must include schema(database) name")
	}
	s, err := ab.Inspector.InspectSchema(ctx, dbName, inspectOptions)
	if err != nil {
		return nil, err
	}
	ab.Types = make(map[string]*gen.Type, len(s.Tables))

	return ab.BuildSchema(s.Tables)
}

func (ab AtlasBase) BuildSchema(tables []*schema.Table) ([]*gen.Type, error) {
	for _, table := range tables {
		if len(ab.Options.Tables) > 0 {
			found := false
			for _, nt := range ab.Options.Tables {
				if nt == table.Name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		//if isJoinTable(table) {
		//	joinTables[table.Name] = table
		//	continue
		//}
		if _, err := ab.TableToGenType(table); err != nil {
			return nil, err
		}
	}
	//for _, table := range tables {
	//	if t, ok := joinTables[table.Name]; ok {
	//		err := upsertManyToMany(m.types, t)
	//		if err != nil {
	//			return nil, err
	//		}
	//		continue
	//	}
	//	m.upsertOneToX(table)
	//}
	ml := make([]*gen.Type, 0, len(tables))
	for _, mutator := range ab.Types {
		ml = append(ml, mutator)
	}
	return ml, nil
}

func (ab AtlasBase) TableToGenType(table *schema.Table) (*gen.Type, error) {
	entSchema := &load.Schema{
		Name:        TypeName(table.Name),
		Annotations: make(map[string]interface{}),
	}
	if entSchema.Name != table.Name {
		ta := entsql.Annotation{Table: table.Name}
		entSchema.Annotations[ta.Name()] = ta
	}
	pk, err := ab.resolvePrimaryKey(table)
	if err != nil {
		return nil, err
	}
	entSchema.Fields = append(entSchema.Fields, pk)
	for i, column := range table.Columns {
		if column.Name == table.PrimaryKey.Parts[0].C.Name {
			continue
		}
		fld, err := ab.resolverField(column)
		fld.Position = &load.Position{Index: i}
		if err != nil {
			return nil, err
		}
		entSchema.Fields = append(entSchema.Fields, fld)
	}
	tb, err := gen.NewType(&gen.Config{}, entSchema)
	if err != nil {
		return nil, err
	}
	ab.Types[table.Name] = tb
	return tb, nil
}

func (ab *AtlasBase) resolvePrimaryKey(table *schema.Table) (f *load.Field, err error) {
	if table.PrimaryKey == nil || len(table.PrimaryKey.Parts) != 1 {
		return nil, fmt.Errorf("entimport: invalid primary key - single part key must be present")
	}
	if f, err = ab.resolverField(table.PrimaryKey.Parts[0].C); err != nil {
		return nil, err
	}
	if f.Name != "id" {
		f.StorageKey = f.Name
		f.Name = "id"
	}
	return f, nil
}
