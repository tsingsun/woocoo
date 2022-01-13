package integration

import (
	"bytes"
	"context"
	"database/sql"
	"github.com/tsingsun/woocoo/cmd/woco/entimport"
	"github.com/tsingsun/woocoo/cmd/woco/entimport/internal/driver"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/go-openapi/inflect"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestMySQL(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	dsn := "root:pass@tcp(localhost:3306)/test?parseTime=True&multiStatements=true"
	db, err := sql.Open(dialect.MySQL, dsn)
	r.NoError(err)
	defer db.Close()
	r.NoError(db.Ping())
	si, err := driver.NewMySQL(driver.WithDSN(dsn))
	r.NoError(err)
	tests := []struct {
		name           string
		query          string
		entities       []string
		expectedFields map[string]string
		expectedEdges  map[string]string
	}{
		{
			name: "one table",
			// language=sql
			query: `
create table users
(
    id   bigint auto_increment primary key,
    age  bigint       not null,
    name varchar(255) not null
);
			`,
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Int("age"), field.String("name")}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return nil
}`,
			},
			entities: []string{"user"},
		},
		{
			name: "int8 and int16 field types",
			// language=sql
			query: `
create table field_type_small_int
(
    id              bigint auto_increment primary key,
    int_8           tinyint  not null,
    int16           smallint not null,
    optional_int8   tinyint null,
    optional_int16  smallint null,
    nillable_int8   tinyint null,
    nillable_int16  smallint null,
    optional_uint8  tinyint unsigned null,
    optional_uint16 smallint unsigned null
);
			`,
			expectedFields: map[string]string{
				"field_type_small_int": `func (FieldTypeSmallInt) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Int8("int_8"), field.Int16("int16"), field.Int8("optional_int8").Optional(), field.Int16("optional_int16").Optional(), field.Int8("nillable_int8").Optional(), field.Int16("nillable_int16").Optional(), field.Uint8("optional_uint8").Optional(), field.Uint16("optional_uint16").Optional()}
}`,
			},
			expectedEdges: map[string]string{
				"field_type_small_int": `func (FieldTypeSmallInt) Edges() []ent.Edge {
	return nil
}`,
			},
			entities: []string{"field_type_small_int"},
		},
		{
			name: "int32 and int64 field types",
			// language=sql
			query: `
create table field_type_int
(
    id                      bigint auto_increment primary key,
    int_field               bigint not null,
    int32                   int    not null,
    int64                   bigint not null,
    optional_int            bigint null,
    optional_int32          int null,
    optional_int64          bigint null,
    nillable_int            bigint null,
    nillable_int32          int null,
    nillable_int64          bigint null,
    validate_optional_int32 int null,
    optional_uint           bigint unsigned null,
    optional_uint32         int unsigned null,
    optional_uint64         bigint unsigned null
);
			`,
			expectedFields: map[string]string{
				"field_type_int": `func (FieldTypeInt) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Int("int_field"), field.Int32("int32"), field.Int("int64"), field.Int("optional_int").Optional(), field.Int32("optional_int32").Optional(), field.Int("optional_int64").Optional(), field.Int("nillable_int").Optional(), field.Int32("nillable_int32").Optional(), field.Int("nillable_int64").Optional(), field.Int32("validate_optional_int32").Optional(), field.Uint64("optional_uint").Optional(), field.Uint32("optional_uint32").Optional(), field.Uint64("optional_uint64").Optional()}
}`,
			},
			expectedEdges: map[string]string{
				"field_type_int": `func (FieldTypeInt) Edges() []ent.Edge {
	return nil
}`,
			},
			entities: []string{"field_type_int"},
		},
		{
			name: "float field types",
			// language=sql
			query: `
create table field_type_float
(
    id              bigint auto_increment primary key,
    float_field     float  not null,
    optional_float  float null,
    double_field    double not null,
    optional_double float null
);
			`,
			expectedFields: map[string]string{
				"field_type_float": `func (FieldTypeFloat) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Float32("float_field"), field.Float32("optional_float").Optional(), field.Float32("double_field"), field.Float32("optional_double").Optional()}
}`,
			},
			expectedEdges: map[string]string{
				"field_type_float": `func (FieldTypeFloat) Edges() []ent.Edge {
	return nil
}`,
			},
			entities: []string{"field_type_float"},
		},
		{
			name: "enum field types",
			// language=sql
			query: `
create table field_type_enum
(
    id                 bigint auto_increment primary key,
    enum_field         enum ('on', 'off') null,
    enum_field_default enum ('ADMIN', 'OWNER', 'USER', 'READ', 'WRITE') default 'READ' not null
);
			`,
			expectedFields: map[string]string{
				"field_type_enum": `func (FieldTypeEnum) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Enum("enum_field").Optional().Values("on", "off"), field.Enum("enum_field_default").Values("ADMIN", "OWNER", "USER", "READ", "WRITE")}
}`,
			},
			expectedEdges: map[string]string{
				"field_type_enum": `func (FieldTypeEnum) Edges() []ent.Edge {
	return nil
}`,
			},
			entities: []string{"field_type_enum"},
		},
		{
			name: "other field types",
			// language=sql
			query: `
create table field_type_other
(
    id               bigint auto_increment primary key,
    datetime         datetime null,
    string           varchar(255) null,
    optional_string  varchar(255) not null,
    bool          tinyint(1) null,
    optional_bool tinyint(1) not null,
    ts               timestamp null
);
			`,
			expectedFields: map[string]string{
				"field_type_other": `func (FieldTypeOther) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Time("datetime").Optional(), field.String("string").Optional(), field.String("optional_string"), field.Bool("bool").Optional(), field.Bool("optional_bool"), field.Time("ts").Optional()}
}`,
			},
			expectedEdges: map[string]string{
				"field_type_other": `func (FieldTypeOther) Edges() []ent.Edge {
	return nil
}`,
			},
			entities: []string{"field_type_other"},
		},
		{
			name: "o2o two types",
			// language=sql
			query: `
create table users
(
    id   bigint auto_increment primary key,
    name varchar(255) not null
);

create table cards
(
    id          bigint auto_increment primary key,
    create_time timestamp not null,
    user_card   bigint null,
    constraint user_card unique (user_card),
    constraint cards_users_card foreign key (user_card) references users (id) on delete set null
);

create index card_id on cards (id);
			`,
			entities: []string{"user", "card"},
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("name")}
}`,
				"card": `func (Card) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Time("create_time"), field.Int("user_card").Optional().Unique()}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return []ent.Edge{edge.To("card", Card.Type).Unique()}
}`,
				"card": `func (Card) Edges() []ent.Edge {
	return []ent.Edge{edge.From("user", User.Type).Ref("card").Unique().Field("user_card")}
}`,
			},
		},
		{
			name: "o2o same type",
			// language=sql
			query: `
create table nodes
(
    id        bigint auto_increment primary key,
    value     bigint null,
    node_next bigint null,
    constraint node_next unique (node_next),
    constraint nodes_nodes_next foreign key (node_next) references nodes (id) on delete set null
);
			`,
			expectedFields: map[string]string{
				"node": `func (Node) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Int("value").Optional(), field.Int("node_next").Optional().Unique()}
}`,
			},
			expectedEdges: map[string]string{
				"node": `func (Node) Edges() []ent.Edge {
	return []ent.Edge{edge.To("child_node", Node.Type).Unique(), edge.From("parent_node", Node.Type).Unique().Field("node_next")}
}`,
			},
			entities: []string{"node"},
		},
		{
			name: "o2o bidirectional",
			// language=sql
			query: `
create table users
(
    id           bigint auto_increment primary key,
    name         varchar(255)                   not null,
    nickname     varchar(255) null,
    user_spouse  bigint null,
    constraint nickname unique (nickname),
    constraint user_spouse unique (user_spouse),
    constraint users_users_spouse foreign key (user_spouse) references users (id) on delete set null
);
			`,
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("name"), field.String("nickname").Optional().Unique(), field.Int("user_spouse").Optional().Unique()}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return []ent.Edge{edge.To("child_user", User.Type).Unique(), edge.From("parent_user", User.Type).Unique().Field("user_spouse")}
}`,
			},
			entities: []string{"user"},
		},
		{
			name: "o2m two types",
			// language=sql
			query: `
create table users
(
    id           bigint auto_increment primary key,
    name         varchar(255)                   not null
);

create table pet
(
    id        bigint auto_increment primary key,
    name      varchar(255)     not null,
    user_pets bigint null,
    constraint pet_users_pets foreign key (user_pets) references users (id) on delete set null
);

create index pet_name_user_pets on pet (name, user_pets);
			`,
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("name")}
}`,
				"pet": `func (Pet) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("name"), field.Int("user_pets").Optional()}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return []ent.Edge{edge.To("pets", Pet.Type)}
}`,
				"pet": `func (Pet) Edges() []ent.Edge {
	return []ent.Edge{edge.From("user", User.Type).Ref("pets").Unique().Field("user_pets")}
}`,
			},
			entities: []string{"user", "pet"},
		},
		{
			name: "o2m same type",
			// language=sql
			query: `
create table users
(
    id           bigint auto_increment
        primary key,
    name         varchar(255)                   not null,
    user_parent  bigint null,
    constraint users_users_parent foreign key (user_parent) references users (id) on delete set null
);
			`,
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("name"), field.Int("user_parent").Optional()}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return []ent.Edge{edge.To("child_users", User.Type), edge.From("parent_user", User.Type).Unique().Field("user_parent")}
}`,
			},
			entities: []string{"user"},
		},
		{
			name: "m2m bidirectional",
			// language=sql
			query: `
create table users
(
    id   bigint auto_increment
        primary key,
    age  bigint       not null,
    name varchar(255) not null
);

create table user_friends
(
    user_id   bigint not null,
    friend_id bigint not null,
    primary key (user_id, friend_id),
    constraint user_friends_friend_id foreign key (friend_id) references users (id) on delete cascade,
    constraint user_friends_user_id foreign key (user_id) references users (id) on delete cascade
);
			`,
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Int("age"), field.String("name")}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return []ent.Edge{edge.To("child_users", User.Type), edge.From("parent_users", User.Type)}
}`,
			},
			entities: []string{"user"},
		},
		{
			name: "m2m same type",
			// language=sql
			query: `
create table users
(
    id   bigint auto_increment primary key,
    name varchar(255) not null
);

create table user_following
(
    user_id     bigint not null,
    follower_id bigint not null,
    primary key (user_id, follower_id),
    constraint user_following_follower_id foreign key (follower_id) references users (id) on delete cascade,
    constraint user_following_user_id foreign key (user_id) references users (id) on delete cascade
);
			`,
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("name")}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return []ent.Edge{edge.To("child_users", User.Type), edge.From("parent_users", User.Type)}
}`,
			},
			entities: []string{"user"},
		},
		{
			// Demonstrate M2M relation between two different types. User and groups.
			name: "m2m two types",
			// language=sql
			query: `
create table some_groups
(
    id        bigint auto_increment primary key,
    active    tinyint(1) default 1 not null,
    name      varchar(255) not null
);

create table users
(
    id           bigint auto_increment primary key,
    name         varchar(255)                   not null
);

create table user_groups
(
    user_id  bigint not null,
    group_id bigint not null,
    primary key (user_id, group_id),
    constraint user_groups_some_groups_id foreign key (group_id) references some_groups (id) on delete cascade,
    constraint user_groups_user_id foreign key (user_id) references users (id) on delete cascade
);
			`,
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("name")}
}`,
				"some_group": `func (SomeGroup) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Bool("active"), field.String("name")}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return []ent.Edge{edge.From("some_groups", SomeGroup.Type).Ref("users")}
}`,
				"some_group": `func (SomeGroup) Edges() []ent.Edge {
	return []ent.Edge{edge.To("users", User.Type)}
}`,
			},
			entities: []string{"user", "some_group"},
		},
		{
			name: "multiple relations",
			// language=sql
			query: `
create table group_infos
(
    id        bigint auto_increment primary key,
    description    varchar(255)         not null,
    max_users bigint default 10000 not null
);

create table some_groups
(
    id         bigint auto_increment primary key,
    name       varchar(255) not null,
    group_info bigint null,
    constraint groups_group_infos_info foreign key (group_info) references group_infos (id) on delete set null
);

create table users
(
    id            bigint auto_increment primary key,
    optional_int  bigint null,
    name          varchar(255)                   not null,
    group_blocked bigint null,
    constraint users_some_groups_blocked foreign key (group_blocked) references some_groups (id) on delete set null
);

create table user_groups
(
    user_id  bigint not null,
    group_id bigint not null,
    primary key (user_id, group_id),
    constraint user_groups_some_groups_id foreign key (group_id) references some_groups (id) on delete cascade,
    constraint user_groups_user_id foreign key (user_id) references users (id) on delete cascade
);
			`,
			expectedFields: map[string]string{
				"user": `func (User) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.Int("optional_int").Optional(), field.String("name"), field.Int("group_blocked").Optional()}
}`,
				"group_info": `func (GroupInfo) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("description"), field.Int("max_users")}
}`,
				"some_group": `func (SomeGroup) Fields() []ent.Field {
	return []ent.Field{field.Int("id"), field.String("name"), field.Int("group_info_id").Optional()}
}`,
			},
			expectedEdges: map[string]string{
				"user": `func (User) Edges() []ent.Edge {
	return []ent.Edge{edge.From("some_groups", SomeGroup.Type).Ref("users"), edge.From("some_group", SomeGroup.Type).Ref("users").Unique().Field("group_blocked")}
}`,
				"group_info": `func (GroupInfo) Edges() []ent.Edge {
	return []ent.Edge{edge.To("some_groups", SomeGroup.Type)}
}`,
				"some_group": `func (SomeGroup) Edges() []ent.Edge {
	return []ent.Edge{edge.From("group_info", GroupInfo.Type).Ref("some_groups").Unique().Field("group_info_id"), edge.To("users", User.Type), edge.To("users", User.Type)}
}`,
			},
			entities: []string{"user", "group_info", "some_group"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dropMySQL(t, db)
			outpath := createTempDir(t)
			_, err := db.ExecContext(ctx, tt.query)
			r.NoError(err)
			schema, err := si.SchemaInspect(ctx)
			r.NoError(err)
			err = entimport.WriteSchema(entimport.SchemaTpl, schema, outpath)
			r.NoError(err)
			r.NotZero(tt.entities)
			actualFiles := readDir(t, outpath)
			r.EqualValues(len(tt.entities), len(actualFiles))
			for _, e := range tt.entities {
				f, err := parser.ParseFile(token.NewFileSet(), "", actualFiles[e+".go"], 0)
				r.NoError(err)
				typeName := inflect.Camelize(e)
				fieldMethod := lookupMethod(f, typeName, "Fields")
				r.NotNil(fieldMethod)
				var actualFields bytes.Buffer
				err = printer.Fprint(&actualFields, token.NewFileSet(), fieldMethod)
				r.NoError(err)
				r.EqualValues(tt.expectedFields[e], actualFields.String())
				edgeMethod := lookupMethod(f, typeName, "Edges")
				r.NotNil(edgeMethod)
				var actualEdges bytes.Buffer
				err = printer.Fprint(&actualEdges, token.NewFileSet(), edgeMethod)
				r.NoError(err)
				r.EqualValues(tt.expectedEdges[e], actualEdges.String())
			}
		})
	}
}

func createTempDir(t *testing.T) string {
	r := require.New(t)
	tmpDir, err := ioutil.TempDir("", "entimport-*")
	r.NoError(err)
	t.Cleanup(func() {
		err = os.RemoveAll(tmpDir)
		r.NoError(err)
	})
	return tmpDir
}

func readDir(t *testing.T, path string) map[string]string {
	files := make(map[string]string)
	err := filepath.Walk(path, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		buf, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.Base(path)] = string(buf)
		return nil
	})
	require.NoError(t, err)
	return files
}

func dropMySQL(t *testing.T, db *sql.DB) {
	r := require.New(t)
	t.Log("drop data from database")
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "DROP DATABASE IF EXISTS test")
	r.NoError(err)
	_, err = db.ExecContext(ctx, "CREATE DATABASE test")
	r.NoError(err)
	_, _ = db.ExecContext(ctx, "USE test")
}

func dropPostgres(t *testing.T, db *sql.DB) {
	r := require.New(t)
	t.Log("drop data from database")
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`)
	r.NoError(err)
}

func lookupMethod(file *ast.File, typeName string, methodName string) (m *ast.FuncDecl) {
	ast.Inspect(file, func(node ast.Node) bool {
		if decl, ok := node.(*ast.FuncDecl); ok {
			if decl.Name.Name != methodName || decl.Recv == nil || len(decl.Recv.List) != 1 {
				return true
			}
			if id, ok := decl.Recv.List[0].Type.(*ast.Ident); ok && id.Name == typeName {
				m = decl
				return false
			}
		}
		return true
	})
	return m
}
