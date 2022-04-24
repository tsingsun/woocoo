package driver

import (
	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/entc/gen"
	"errors"
	"github.com/go-openapi/inflect"
)

const (
	to edgeDir = iota
	from

	FuncTag = "func:"
)

type (
	edgeDir int
	options struct {
		uniqueEdgeToChild    bool
		recursive            bool
		uniqueEdgeFromParent bool
		refName              string
		edgeField            string
	}
)

var joinTableErr = errors.New("entimport: join tables must be inspected with ref tables - append `tables` flag")

func entEdge(nodeName, nodeType string, currentNode *gen.Type, dir edgeDir, opts options) (e *gen.Edge) {
	e = &gen.Edge{
		Name: inflect.Singularize(nodeName),
	}
	switch dir {
	case to:
		//ed := edge.To(nodeName, ent.Schema.Type)
		if opts.uniqueEdgeToChild {
			e.Unique = true
		}
		if opts.recursive {
			e.Name = "child_" + e.Name
		}
	case from:
		//ed := edge.From(nodeName, ent.Schema.Type)
		if opts.uniqueEdgeFromParent {
			e.Unique = true
		}
		if opts.edgeField != "" {
			setEdgeField(e, opts, currentNode)
		}
		if opts.recursive {
			e.Name = "parent_" + e.Name
			break
		}
		// RefName describes which entEdge of the Parent Node we're referencing
		// because there can be multiple references from one node to another.
		refName := opts.refName
		if opts.uniqueEdgeToChild {
			refName = inflect.Singularize(refName)
		}
		e.Ref.Name = refName
	}
	e.Type = currentNode
	return e
}

func setEdgeField(e *gen.Edge, opts options, childNode *gen.Type) {
	edgeField := opts.edgeField
	if e.Name == edgeField {
		edgeField += "_id"
		for _, f := range childNode.Fields {
			if f.Name == opts.edgeField {
				f.Name = edgeField
			}
		}
	}
	e.Field().Name = edgeField
}

func upsertRelation(nodeA *gen.Type, nodeB *gen.Type, opts options) {
	tableA := TableName(nodeA.Name)
	tableB := TableName(nodeB.Name)
	opts.refName = tableB
	fromA := entEdge(tableA, nodeA.Name, nodeB, from, opts)
	toB := entEdge(tableB, nodeB.Name, nodeA, to, opts)
	nodeA.Edges = append(nodeA.Edges, toB)
	nodeB.Edges = append(nodeB.Edges, fromA)
}

func upsertManyToMany(types map[string]*gen.Type, table *schema.Table) error {
	tableA := table.ForeignKeys[0].RefTable
	tableB := table.ForeignKeys[1].RefTable
	var opts options
	if tableA.Name == tableB.Name {
		opts.recursive = true
	}
	nodeA, ok := types[tableA.Name]
	if !ok {
		return joinTableErr
	}
	nodeB, ok := types[tableA.Name]
	if !ok {
		return joinTableErr
	}
	upsertRelation(nodeA, nodeB, opts)
	return nil
}

// Note: at this moment ent doesn't support fields on m2m relations.
func isJoinTable(table *schema.Table) bool {
	if table.PrimaryKey == nil || len(table.PrimaryKey.Parts) != 2 || len(table.ForeignKeys) != 2 {
		return false
	}
	// Make sure that the foreign key columns exactly match primary key column.
	for _, fk := range table.ForeignKeys {
		if len(fk.Columns) != 1 {
			return false
		}
		if fk.Columns[0] != table.PrimaryKey.Parts[0].C && fk.Columns[0] != table.PrimaryKey.Parts[1].C {
			return false
		}
	}
	return true
}

func TypeName(tableName string) string {
	return inflect.Camelize(inflect.Singularize(tableName))
}

func TableName(typeName string) string {
	return inflect.Underscore(inflect.Pluralize(typeName))
}
