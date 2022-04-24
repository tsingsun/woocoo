package entimport

import (
	"entgo.io/ent/dialect"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/entimport/internal/driver"
	"strings"
	"text/template"
)

var Funcs = template.FuncMap{
	"schemaType":       schemaType,
	"fieldTypeName":    fieldTypeName,
	"clearComment":     clearComment,
	"entgqlOrderField": entgqlOrderField,
	"defaultValue":     defaultValue,
	"fieldParameters":  fieldParameters,
	"inc": func(i int) int {
		return i + 1
	},
}

func schemaType(f *gen.Field) string {
	col := f.Column()
	var builder strings.Builder
	if col.SchemaType != nil {
		builder.WriteString(".SchemaType(map[string]string{")
		for k, v := range col.SchemaType {
			var st = `"` + k + `"`
			if k == dialect.MySQL {
				st = "dialect.MySQL"
			} else if k == dialect.Postgres {
				st = "dialect.Postgres"
			} else if k == dialect.SQLite {
				st = "dialect.SQLite"
			} else if k == dialect.Gremlin {
				st = "dialect.Gremlin"
			}
			builder.WriteString(fmt.Sprintf("%s:\"%s\",", st, v))
		}
		val := builder.String()
		v := val[0 : len(val)-1]
		return v + "})"
	}
	return builder.String()
}

// fieldTypeName return Field.XXX() for generating Field() method
func fieldTypeName(f *gen.Field) string {
	if f.Type.Type.Float() {
		if f.Type.Type == field.TypeFloat64 {
			return "Float"
		}
	} else if f.IsTime() {
		return "Time"
	}
	if rt := f.Type.RType; rt != nil {
		return rt.Name
	}
	return f.Type.String()
}

func fieldParameters(f *gen.Field) string {
	switch f.Type.Type {
	case field.TypeUUID:
		return fmt.Sprintf(`"%s",%s{}`, f.Name, f.Type.Ident)
	default:
		return `"` + f.Name + `"`
	}
}

func clearComment(f *gen.Field) string {
	c := f.Comment()
	v := strings.ReplaceAll(c, "\\r", "")
	v = strings.ReplaceAll(v, "\\n", "")
	return v
}

func entgqlOrderField(f *gen.Field) bool {
	for _, i := range f.Annotations {
		_, ok := i.(*driver.GqlOrderField)
		if ok {
			return true
		}
	}
	return false
}

func cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

func defaultValue(f *gen.Field) string {
	switch v := f.DefaultValue().(type) {
	case string:
		_, str, ok := cut(v, driver.FuncTag)
		if ok {
			return str
		}
		return `"` + f.DefaultValue().(string) + `"`
	default:
		return ""
	}
}
