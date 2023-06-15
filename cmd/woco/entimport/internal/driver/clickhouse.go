package driver

import (
	"ariga.io/atlas/sql/schema"
	"context"
	"database/sql"
	"encoding/json"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/entc/load"
	"entgo.io/ent/schema/field"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/google/uuid"
	"strconv"
	"strings"
	"time"
)

const (
	// Query to list database schemas.
	schemasQuery     = "SELECT name from system.databases WHERE name NOT IN ('mysql', 'information_schema', 'INFORMATION_SCHEMA', 'system') ORDER BY name"
	schemasQueryArgs = "SELECT name from system.databases WHERE name %s"
	tablesQuery      = "select database,name,primary_key,sorting_key FROM system.tables where database in (%s) order by database,name"
	tablesQueryArgs  = "select database,name,primary_key,sorting_key FROM system.tables where database in (%s) and name in (%s) order by database,name"
	columnsQuery     = "select table,name,type,comment,default_expression,is_in_sorting_key,is_in_primary_key FROM system.columns where database = $1 and table in (%s) order by database,table,position"
)

const (
	Nullable = "Nullable"
)

type Clickhouse struct {
	AtlasBase
	types map[string]*gen.Type
}

type ckDriver struct {
	schema.ExecQuerier
}

// NewClickhouse creates a new Clickhouse driver.
func NewClickhouse(opts ImportOptions) (*Clickhouse, error) {
	copts, err := clickhouse.ParseDSN(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("entimport: failed to parse DSN for clickhouse connection: %w", err)
	}
	db, err := sql.Open("clickhouse", opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("entimport: error while trying to open db inspection client %w", err)
	}
	drv, err := OpenCK(db)
	if err != nil {
		return nil, fmt.Errorf("entimport: error while trying to open db inspection client %w", err)
	}
	ab := AtlasBase{
		Options:   opts,
		Inspector: drv,
		resolverDsn: func(dsn string) (db string, err error) {
			return copts.Auth.Database, nil
		},
	}
	dr := &Clickhouse{
		AtlasBase: ab,
	}
	dr.AtlasBase.resolverField = dr.resolverField
	return dr, nil
}

func (ck *ckDriver) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Schema, error) {
	schemas, err := ck.schemas(ctx, &schema.InspectRealmOption{Schemas: []string{name}})
	if err != nil {
		return nil, err
	}
	switch n := len(schemas); {
	case n == 0:
		return nil, &schema.NotExistError{Err: fmt.Errorf("clickhouse: schema %q was not found", name)}
	case n > 1:
		found := false
		for _, s := range schemas {
			if s.Name == name {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("clickhouse: %d schemas were found for %q", n, name)
		}
	}
	r := schema.NewRealm(schemas...)
	if err := ck.inspectTables(ctx, r, opts); err != nil {
		return nil, err
	}
	return r.Schemas[0], nil
}

func (ck *ckDriver) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	//TODO implement me
	panic("implement me")
}

func (ck *ckDriver) schemas(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var (
		args  []any
		query = schemasQuery
	)
	if opts != nil {
		switch n := len(opts.Schemas); {
		case n == 1 && opts.Schemas[0] == "":
		case n == 1 && opts.Schemas[0] != "":
			query = fmt.Sprintf(schemasQueryArgs, "= $1")
			args = append(args, opts.Schemas[0])
		case n > 0:
			for _, s := range opts.Schemas {
				args = append(args, s)
			}
			query = fmt.Sprintf(schemasQueryArgs, fmt.Sprintf("IN (%s)", nArgs(2, len(opts.Schemas)+1)))
		}
	}
	rows, err := ck.QueryContext(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: querying schemas: %w", err)
	}
	defer rows.Close()
	var schemas []*schema.Schema
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, &schema.Schema{
			Name:  name,
			Attrs: []schema.Attr{},
		})
	}
	return schemas, nil
}

func (ck *ckDriver) inspectTables(ctx context.Context, r *schema.Realm, opts *schema.InspectOptions) error {
	if err := ck.tables(ctx, r, opts); err != nil {
		return err
	}
	for _, s := range r.Schemas {
		if len(s.Tables) == 0 {
			continue
		}
		if err := ck.columns(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func nArgs(start, end int) string {
	args := make([]string, end-start)
	for i := start; i <= end; i++ {
		args = append(args, "$"+strconv.Itoa(i))
	}
	return strings.Join(args, ",")
}

func (ck *ckDriver) tables(ctx context.Context, realm *schema.Realm, opts *schema.InspectOptions) error {
	var (
		args  []any
		query = fmt.Sprintf(tablesQuery, nArgs(1, len(realm.Schemas)))
	)
	for _, s := range realm.Schemas {
		args = append(args, s.Name)
	}
	if opts != nil && len(opts.Tables) > 0 {
		for _, t := range opts.Tables {
			args = append(args, t)
		}
		query = fmt.Sprintf(tablesQueryArgs, nArgs(1, len(realm.Schemas)), nArgs(len(realm.Schemas)+1, len(opts.Tables)+1))
	}
	rows, err := ck.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			tSchema, name, primaryKyes, sortKeys sql.NullString
		)
		if err := rows.Scan(&tSchema, &name, &primaryKyes, &sortKeys); err != nil {
			return fmt.Errorf("scan table information: %w", err)
		}
		s, ok := realm.Schema(tSchema.String)
		if !ok {
			return fmt.Errorf("schema %q was not found in realm", tSchema.String)
		}
		t := &schema.Table{Name: name.String}
		if primaryKyes.Valid && primaryKyes.String != "" {

		}
		if sortKeys.Valid && sortKeys.String != "" {

		}
		// need version supported
		//if comment.Valid {
		//	t.Attrs = append(t.Attrs, &schema.Comment{
		//		Text: comment.String,
		//	})
		//}
		s.AddTables(t)
	}
	return rows.Close()
}

// columns queries and appends the columns of the given table.
func (ck *ckDriver) columns(ctx context.Context, s *schema.Schema) error {
	args := []any{s.Name}
	for _, t := range s.Tables {
		args = append(args, t.Name)
	}
	rows, err := ck.QueryContext(ctx, fmt.Sprintf(columnsQuery, nArgs(2, len(s.Tables)+1)), args...)
	if err != nil {
		return fmt.Errorf("clickhouse: query schema %q columns: %w", s.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := ck.addColumn(s, rows); err != nil {
			return fmt.Errorf("mysql: %w", err)
		}
	}
	return rows.Err()
}

func (ck *ckDriver) querySchema(ctx context.Context, query string, s *schema.Schema) (*sql.Rows, error) {
	args := []any{s.Name}
	for _, t := range s.Tables {
		args = append(args, t.Name)
	}
	return ck.QueryContext(ctx, fmt.Sprintf(query, nArgs(1, len(s.Tables))), args...)
}

// addColumn scans the current row and adds a new column from it to the table.
func (ck *ckDriver) addColumn(s *schema.Schema, rows *sql.Rows) error {
	var table, name, typ, comment, defaultExpression, isInSortingKey, isInPrimaryKey sql.NullString
	if err := rows.Scan(&table, &name, &typ, &comment, &defaultExpression, &isInSortingKey, &isInPrimaryKey); err != nil {
		return err
	}
	t, ok := s.Table(table.String)
	if !ok {
		return fmt.Errorf("table %q was not found in schema", table.String)
	}
	c := &schema.Column{
		Name: name.String,
		Type: &schema.ColumnType{
			Raw:  typ.String,
			Null: strings.Contains(typ.String, Nullable),
		},
	}
	ct, err := ParseType(c.Type.Raw)
	if err != nil {
		return err
	}
	c.Type.Type = ct
	if defaultExpression.Valid && defaultExpression.String != "" {
		c.Default = &schema.RawExpr{
			X: defaultExpression.String,
		}
	}
	if comment.Valid && comment.String != "" {
		c.Attrs = append(c.Attrs, &schema.Comment{
			Text: comment.String,
		})
	}
	if isInSortingKey.Valid && isInSortingKey.String == "1" {
		idx := &schema.Index{
			Name:  c.Name,
			Table: t,
		}
		idx.Parts = append(idx.Parts, &schema.IndexPart{
			C:     c,
			SeqNo: len(idx.Parts) + 1,
		})
		c.Indexes = append(c.Indexes, idx)
		//t.AddIndexes(idx)
	}
	t.Columns = append(t.Columns, c)

	if c.Type.Raw == "UUID" {
		if t.PrimaryKey == nil {
			t.PrimaryKey = &schema.Index{Table: t, Name: "id"}
		}
		t.PrimaryKey.Parts = append(t.PrimaryKey.Parts, &schema.IndexPart{
			C:     c,
			SeqNo: len(t.PrimaryKey.Parts) + 1,
		})
	}

	return nil
}

func baseType(raw string) string {
	rr := ""
	switch start, end := strings.Index(raw, "("), strings.LastIndex(raw, ")"); {
	case len(raw) == 0, start <= 0, end <= 0, end < start:
		rr = raw
	default:
		rr = raw[start+1 : end]
	}
	srr := strings.SplitN(rr, ",", 2)
	if len(srr) > 1 {
		nstr := strings.TrimSpace(srr[1])
		if strings.HasPrefix(nstr, Nullable+"(") {
			return nstr[len(Nullable+"(") : len(nstr)-1]
		}
		return nstr
	}
	return rr
}

// ParseType returns the schema.Type value represented by the given raw type.
// The raw value is expected to follow the format in MySQL information schema.
func ParseType(raw string) (schema.Type, error) {
	base := baseType(raw)
	ckType, err := column.Type(base).Column("", time.Local) // column's parameter is not used
	if err != nil {
		return nil, err
	}

	switch ckt := ckType.(type) {
	case *column.String:
		return &schema.StringType{
			T: string(ckt.Type()),
		}, nil
	// bool and booleans are synonyms for
	// tinyint with display-width set to 1.
	case *column.Bool:
		return &schema.BoolType{
			T: string(ckt.Type()),
		}, nil
	case *column.Int8, *column.Int16, *column.Int32, *column.Int64:
		ft := &schema.IntegerType{
			T:        string(ckType.Type()),
			Unsigned: false,
		}
		return ft, nil
	case *column.UInt8, *column.UInt16, *column.UInt32, *column.UInt64:
		ft := &schema.IntegerType{
			T:        string(ckType.Type()),
			Unsigned: true,
		}
		return ft, nil
	case *column.Decimal:
		dt := &schema.DecimalType{
			T:        string(ckt.Type()),
			Unsigned: false,
		}
		dt.Precision = int(ckt.Precision())
		dt.Scale = int(ckt.Scale())
		return dt, nil
	case *column.Float32, *column.Float64:
		ft := &schema.FloatType{
			T:        string(ckType.Type()),
			Unsigned: false,
		}
		return ft, nil
	case *column.FixedString:
		var size int
		if _, err := fmt.Sscanf(raw, "FixedString(%d)", &size); err != nil {
			return nil, err
		}

		return &schema.StringType{
			T:    string(ckt.Type()),
			Size: size,
		}, nil
	case *column.Date, *column.Date32, *column.DateTime, *column.DateTime64:
		tt := &schema.TimeType{
			T: string(ckType.Type()),
		}
		return tt, nil
	case *column.UUID:
		tt := &schema.StringType{
			T: string(ckt.Type()),
		}
		return tt, nil
	default:
		return &schema.UnsupportedType{
			T: string(ckType.Type()),
		}, nil
	}
}

func OpenCK(db schema.ExecQuerier) (*ckDriver, error) {
	ckDriver := &ckDriver{
		ExecQuerier: db,
	}
	return ckDriver, nil
}

func (ck *Clickhouse) resolverField(column *schema.Column) (f *load.Field, err error) {
	name := column.Name
	fd := &field.Descriptor{
		Name: name,
	}
	fd.Optional = column.Type.Null
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
		fd.Info = field.Float(name).Descriptor().Info
	case *schema.IntegerType:
		fd.Info = ck.convertInteger(typ, name)
	case *schema.JSONType:
		fd.Info = field.JSON(name, json.RawMessage{}).Descriptor().Info
	case *schema.StringType:
		switch typ.T {
		case "UUID":
			fd.Info = field.UUID(name, uuid.UUID{}).Descriptor().Info
		default:
			em := field.String(name).MaxLen(column.Type.Type.(*schema.StringType).Size)
			fd.Info = em.Descriptor().Info
			fd.Size = em.Descriptor().Size
		}
	case *schema.TimeType:
		fd.Info = field.Time(name).Descriptor().Info
	default:
		return nil, fmt.Errorf("entimport: unsupported type %q", typ)
	}
	ck.applyColumnAttributes(fd, column)
	return load.NewField(fd)
}

func (ck *Clickhouse) convertInteger(typ *schema.IntegerType, name string) (f *field.TypeInfo) {
	if ck.Options.CaseInt {
		f = field.Int(name).Descriptor().Info
		return f
	}
	if typ.Unsigned {
		switch typ.T {
		case string((&column.UInt8{}).Type()):
			f = field.Uint8(name).Descriptor().Info
		case string((&column.UInt16{}).Type()):
			f = field.Uint16(name).Descriptor().Info
		case string((&column.UInt32{}).Type()):
			f = field.Uint32(name).Descriptor().Info
		case string((&column.UInt64{}).Type()):
			f = field.Uint64(name).Descriptor().Info
		}
		return f
	}
	switch typ.T {
	case string((&column.Int8{}).Type()):
		f = field.Int8(name).Descriptor().Info
	case string((&column.Int16{}).Type()):
		f = field.Int16(name).Descriptor().Info
	case string((&column.Int32{}).Type()):
		f = field.Int32(name).Descriptor().Info
	case string((&column.Int64{}).Type()):
		f = field.Int32(name).Descriptor().Info
	}
	return f
}

func (ck *Clickhouse) applyColumnAttributes(f *field.Descriptor, col *schema.Column) {
	switch dt := col.Default.(type) {
	case *schema.Literal:
		f.Default = dt.V
	case *schema.RawExpr:
		if dt.X == "generateUUIDv4()" {
			f.Default = FuncTag + "uuid.New"
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
