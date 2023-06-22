package entimport

import (
	"bytes"
	"context"
	"embed"
	"entgo.io/ent/entc/gen"
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/entimport/internal/driver"
	"os"
	"path/filepath"
	"strings"

	"entgo.io/ent/dialect"
)

var (
	//go:embed template/*
	templates embed.FS
	SchemaTpl = parseT("template/schema.tmpl")
	GqlTpl    = parseT("template/graphql.tmpl")
)

func generateSchema(opts driver.ImportOptions) error {
	ctx := context.Background()
	i, err := NewImport(opts)
	if err != nil {
		return fmt.Errorf("entimport: create importer (%s) failed - %v", opts.Dialect, err)
	}
	schema, err := i.SchemaInspect(ctx)
	if err != nil {
		return fmt.Errorf("entimport: schema import failed - %v", err)
	}

	if err = WriteSchema(SchemaTpl, schema, opts); err != nil {
		return fmt.Errorf("entimport: schema writing failed - %v", err)
	}
	if opts.GenGraphql {
		if err = WriteGql(GqlTpl, schema, opts.SchemaPath); err != nil {
			return fmt.Errorf("entimport: graphql writing failed - %v", err)
		}
	}
	return nil
}

func parseT(path string) *gen.Template {
	return gen.MustParse(gen.NewTemplate(path).
		Funcs(gen.Funcs).
		Funcs(Funcs).
		ParseFS(templates, path))
}

// SchemaImporter is the interface that wraps the Schema.
type SchemaImporter interface {
	// SchemaInspect imports a given schema from a data source and returns a list of gen.Type.
	SchemaInspect(context.Context) ([]*gen.Type, error)
}

// NewImport - calls the relevant data source importer based on a given dialect.
func NewImport(opts driver.ImportOptions) (SchemaImporter, error) {
	var (
		si  SchemaImporter
		err error
	)
	switch opts.Dialect {
	case dialect.MySQL:
		si, err = driver.NewMySQL(opts)
		if err != nil {
			return nil, err
		}
	case dialect.Postgres:
		//si, err = NewPostgreSQL(opts...)
		//if err != nil {
		//	return nil, err
		//}
	case "clickhouse":
		si, err = driver.NewClickhouse(opts)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("entimport: unsupported dialect %q", opts.Dialect)
	}
	return si, err
}

func createDir(target string) error {
	_, err := os.Stat(target)
	if err == nil || !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		return fmt.Errorf("creating schema directory: %w", err)
	}
	return nil
}

func WriteSchema(template *gen.Template, types []*gen.Type, opts driver.ImportOptions) error {
	if err := createDir(opts.SchemaPath); err != nil {
		return fmt.Errorf("create dir %s: %w", opts.SchemaPath, err)
	}
	for _, typ := range types {
		typ.Annotations["proto"] = opts.GenProtoField
		typ.Annotations["gql"] = opts.GenGraphql
		name := typ.Name
		if err := gen.ValidSchemaName(typ.Name); err != nil {
			return fmt.Errorf("init schema %s: %w", typ.Name, err)
		}
		b := bytes.NewBuffer(nil)
		if err := template.ExecuteTemplate(b, "schema.tmpl", typ); err != nil {
			return fmt.Errorf("executing template %s: %w", name, err)
		}
		newFileTarget := filepath.Join(opts.SchemaPath, strings.ToLower(name+".go"))
		if err := os.WriteFile(newFileTarget, b.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing file %s: %w", newFileTarget, err)
		}
	}
	return nil
}

func WriteGql(template *gen.Template, types []*gen.Type, output string) error {
	if err := createDir(output); err != nil {
		return fmt.Errorf("create dir %s: %w", output, err)
	}
	for _, typ := range types {
		name := typ.Name
		if err := gen.ValidSchemaName(typ.Name); err != nil {
			return fmt.Errorf("init schema %s: %w", typ.Name, err)
		}
		b := bytes.NewBuffer(nil)
		if err := template.ExecuteTemplate(b, "graphql.tmpl", typ); err != nil {
			return fmt.Errorf("executing template %s: %w", name, err)
		}
		newFileTarget := filepath.Join(output, strings.ToLower(name+".graphql"))
		if err := os.WriteFile(newFileTarget, b.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing file %s: %w", newFileTarget, err)
		}
	}
	return nil
}
