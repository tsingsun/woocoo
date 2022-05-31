package driver

import (
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
	"database/sql"
	"encoding/json"
	"entgo.io/ent/dialect"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/entc/load"
	"entgo.io/ent/schema/field"
	"fmt"
	mysqldriver "github.com/go-sql-driver/mysql"
	"strings"
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
	AtlasBase
	types map[string]*gen.Type
}

// NewMySQL - create an import structure for MySQL.
func NewMySQL(opts ImportOptions) (*MySQL, error) {
	db, err := sql.Open(dialect.MySQL, opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("entimport: failed to open db connection: %w", err)
	}
	drv, err := mysql.Open(db)
	if err != nil {
		return nil, fmt.Errorf("entimport: error while trying to open db inspection client %w", err)
	}
	ab := AtlasBase{
		Options:   opts,
		Inspector: drv,
		resolverDsn: func(dsn string) (db string, err error) {
			cfg, err := mysqldriver.ParseDSN(dsn)
			return cfg.DBName, err
		},
	}
	dr := &MySQL{
		AtlasBase: ab,
	}
	dr.AtlasBase.resolverField = dr.resolverField
	return dr, nil
}

func (m *MySQL) resolverField(column *schema.Column) (f *load.Field, err error) {
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
		if m.Options.CaseInt {
			fd.Info = field.Int(name).Descriptor().Info
			//resolverField.Int is big int
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
		//TODO literal use "" for default value
		if strings.HasPrefix(dt.V, "\"") {
			f.Default = dt.V[1 : len(dt.V)-1]
		}
		f.Default = dt.V
	case *schema.RawExpr:
		if strings.ToLower(dt.X) == "current_timestamp()" || strings.ToLower(dt.X) == "current_timestamp" {
			f.Default = FuncTag + "time.Now"
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

		f.Annotations = append(f.Annotations, &GqlOrderField{
			name: col.Name,
		})

	}
	// FK / Reference column
	if col.ForeignKeys != nil {
		f.Optional = true
	}
}
