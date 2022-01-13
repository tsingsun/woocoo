package driver

import (
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
	"context"
	"database/sql"
	"encoding/json"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/entc/load"
	"entgo.io/ent/schema/field"
	"errors"
	"fmt"
	"github.com/go-openapi/inflect"
	mysqldriver "github.com/go-sql-driver/mysql"
	"strings"
	"time"
)

const (
	mTinyInt   = "tinyint"   // MYSQL_TYPE_TINY
	mSmallInt  = "smallint"  // MYSQL_TYPE_SHORT
	mInt       = "int"       // MYSQL_TYPE_LONG
	mMediumInt = "mediumint" // MYSQL_TYPE_INT24
	mBigInt    = "bigint"    // MYSQL_TYPE_LONGLONG
)

// MySQL holds the schema import options and an Atlas inspector instance
type MySQL struct {
	schema.Inspector
	Options *ImportOptions
	types   map[string]*gen.Type
}

// NewMySQL - create aמ import structure for MySQL.
func NewMySQL(opts ...ImportOption) (*MySQL, error) {
	i := &ImportOptions{
		caseInt: true,
	}
	for _, apply := range opts {
		apply(i)
	}
	db, err := sql.Open(dialect.MySQL, i.dsn)
	if err != nil {
		return nil, fmt.Errorf("entimport: failed to open db connection: %w", err)
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, fmt.Errorf("entimport: error while trying to open db inspection client %w", err)
	}
	return &MySQL{
		Inspector: drv,
		Options:   i,
	}, nil
}

// SchemaInspect implements SchemaImporter.
func (m *MySQL) SchemaInspect(ctx context.Context) ([]*gen.Type, error) {
	inspectOptions := &schema.InspectOptions{
		Tables: m.Options.tables,
	}
	// dsn example: root:pass@tcp(localhost:3308)/test?parseTime=True
	cfg, err := mysqldriver.ParseDSN(m.Options.dsn)
	if err != nil {
		return nil, err
	}
	if cfg.DBName == "" {
		return nil, errors.New("DSN connection string must include schema(database) name")
	}
	s, err := m.Inspector.InspectSchema(ctx, cfg.DBName, inspectOptions)
	if err != nil {
		return nil, err
	}
	m.types = make(map[string]*gen.Type, len(s.Tables))

	return m.buildSchema(s.Tables)
}

func (m *MySQL) buildSchema(tables []*schema.Table) ([]*gen.Type, error) {
	//joinTables := make(map[string]*schema.Table)
	for _, table := range tables {
		if len(m.Options.tables) > 0 {
			found := false
			for _, nt := range m.Options.tables {
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
		if _, err := m.upsertNode(table); err != nil {
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
	for _, mutator := range m.types {
		ml = append(ml, mutator)
	}
	return ml, nil
}

func (m *MySQL) resolvePrimaryKey(table *schema.Table) (f *load.Field, err error) {
	if table.PrimaryKey == nil || len(table.PrimaryKey.Parts) != 1 {
		return nil, fmt.Errorf("entimport: invalid primary key - single part key must be present")
	}
	if f, err = m.field(table.PrimaryKey.Parts[0].C); err != nil {
		return nil, err
	}
	if f.Name != "id" {
		f.StorageKey = f.Name
		f.Name = "id"
	}
	if m.Options.caseInt {
		f.SchemaType = map[string]string{dialect.MySQL: "INT"}
	}
	return f, nil
}

func (m *MySQL) field(column *schema.Column) (f *load.Field, err error) {
	name := column.Name
	fd := &field.Descriptor{
		Name: name,
	}
	switch typ := column.Type.Type.(type) {
	case *schema.BinaryType:
		fd.Info = field.Bytes(name).Descriptor().Info
	case *schema.BoolType:
		fd.Info = field.Bool(name).Descriptor().Info
	case *schema.DecimalType:
		fd.Info = field.Float(name).Descriptor().Info
	case *schema.EnumType:
		em := field.Enum(name).Values(typ.Values...)
		fd.Info = em.Descriptor().Info
		fd.Enums = em.Descriptor().Enums
	case *schema.FloatType:
		fd.Info = m.convertFloat(typ, name)
	case *schema.IntegerType:
		if m.Options.caseInt {
			fd.Info = field.Int(name).Descriptor().Info
			//field.Int is big int
			if typ.T != mBigInt {
				fd.SchemaType = map[string]string{dialect.MySQL: strings.ToUpper(typ.T)}
			}
		} else {
			fd.Info = m.convertInteger(typ, name)
		}
	case *schema.JSONType:
		fd.Info = field.JSON(name, json.RawMessage{}).Descriptor().Info
	case *schema.StringType:
		em := field.String(name).MaxLen(column.Type.Type.(*schema.StringType).Size)
		fd.Info = em.Descriptor().Info
		fd.Size = em.Descriptor().Size
	case *schema.TimeType:
		fd.Info = field.Time(name).Descriptor().Info
	default:
		return nil, fmt.Errorf("entimport: unsupported type %q", typ)
	}
	m.applyColumnAttributes(fd, column)
	return load.NewField(fd)
}

func (m *MySQL) convertFloat(typ *schema.FloatType, name string) (f *field.TypeInfo) {
	// A precision from 0 to 23 results in a 4-byte single-precision FLOAT column.
	// A precision from 24 to 53 results in an 8-byte double-precision DOUBLE column:
	// https://dev.mysql.com/doc/refman/8.0/en/floating-point-types.html

	if typ.Precision > 23 {
		return field.Float(name).Descriptor().Info
	}

	return field.Float32(name).Descriptor().Info
}

func (m *MySQL) convertInteger(typ *schema.IntegerType, name string) (f *field.TypeInfo) {
	if typ.Unsigned {
		switch typ.T {
		case mTinyInt:
			f = field.Uint8(name).Descriptor().Info
		case mSmallInt:
			f = field.Uint16(name).Descriptor().Info
		case mMediumInt:
			f = field.Uint32(name).Descriptor().Info
		case mInt:
			f = field.Uint32(name).Descriptor().Info
		case mBigInt:
			f = field.Uint64(name).Descriptor().Info
		}
		return f
	}
	switch typ.T {
	case mTinyInt:
		f = field.Int8(name).Descriptor().Info
	case mSmallInt:
		f = field.Int16(name).Descriptor().Info
	case mMediumInt:
		f = field.Int32(name).Descriptor().Info
	case mInt:
		f = field.Int32(name).Descriptor().Info
	case mBigInt:
		// Int64 is not used on purpose.
		f = field.Int(name).Descriptor().Info
	}
	return f
}

func (m *MySQL) applyColumnAttributes(f *field.Descriptor, col *schema.Column) {
	f.Optional = col.Type.Null
	switch dt := col.Default.(type) {
	case *schema.Literal:
		f.Default = dt.V
	case *schema.RawExpr:
		if dt.X == "current_timestamp()" {
			f.Default = time.Now
		}
	}
	for _, attr := range col.Attrs {
		if a, ok := attr.(*schema.Comment); ok {
			f.Comment = a.Text
		}
	}
	for _, idx := range col.Indexes {
		if idx.Unique && len(idx.Parts) == 1 {
			f.Unique = idx.Unique
		}
	}
	// FK / Reference column
	if col.ForeignKeys != nil {
		f.Optional = true
	}
}

// O2O Two Types - Child Table has a unique reference (FK) to Parent table
// O2O Same Type - Child Table has a unique reference (FK) to Parent table (itself)
// O2M (The "Many" side, keeps a reference to the "One" side).
// O2M Two Types - Parent has a non-unique reference to Child, and Child has a unique back-reference to Parent
// O2M Same Type - Parent has a non-unique reference to Child, and Child doesn't have a back-reference to Parent.
func (m *MySQL) upsertOneToX(table *schema.Table) {
	if table.ForeignKeys == nil || table.Indexes == nil {
		return
	}
	for _, fk := range table.ForeignKeys {
		if len(fk.Columns) != 1 {
			continue
		}
		for _, idx := range table.Indexes {
			if len(idx.Parts) != 1 {
				continue
			}
			// MySQL requires indexes on foreign keys and referenced keys.
			if fk.Columns[0] == idx.Parts[0].C {
				parent := fk.RefTable
				child := table
				opts := options{
					uniqueEdgeFromParent: true,
					refName:              child.Name,
				}
				if child.Name == parent.Name {
					opts.recursive = true
				}
				if idx.Unique {
					opts.uniqueEdgeToChild = true
				}
				opts.edgeField = fk.Columns[0].Name
				// If at least one table in the relation does not exist, there is no point to create it.
				parentNode, ok := m.types[parent.Name]
				if !ok {
					return
				}
				childNode, ok := m.types[child.Name]
				if !ok {
					return
				}
				upsertRelation(parentNode, childNode, opts)
			}
		}
	}
}

func (m *MySQL) upsertNode(table *schema.Table) (*gen.Type, error) {
	upsert := &load.Schema{
		Name:        typeName(table.Name),
		Annotations: make(map[string]interface{}),
	}
	if upsert.Name != table.Name {
		ta := entsql.Annotation{Table: table.Name}
		upsert.Annotations[ta.Name()] = ta
	}

	pk, err := m.resolvePrimaryKey(table)
	if err != nil {
		return nil, err
	}
	upsert.Fields = append(upsert.Fields, pk)

	for i, column := range table.Columns {
		if column.Name == table.PrimaryKey.Parts[0].C.Name {
			continue
		}
		fld, err := m.field(column)
		fld.Position = &load.Position{Index: i}
		if err != nil {
			return nil, err
		}
		upsert.Fields = append(upsert.Fields, fld)
	}
	tb, err := gen.NewType(&gen.Config{}, upsert)
	if err != nil {
		return nil, err
	}
	m.types[table.Name] = tb
	return tb, nil
}

func typeName(tableName string) string {
	return inflect.Camelize(inflect.Singularize(tableName))
}

func tableName(typeName string) string {
	return inflect.Underscore(inflect.Pluralize(typeName))
}
