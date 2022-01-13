package entimport

import (
	"entgo.io/ent/dialect"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"fmt"
	"strings"
	"text/template"
)

var Funcs = template.FuncMap{
	"schemaType":    schemaType,
	"fieldTypeName": fieldTypeName,
	"clearComment":  clearComment,
}

func schemaType(f gen.Field) string {
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
		builder.WriteString("})")
	}
	return builder.String()
}

// fieldTypeName return Field.XXX() for generating Field() method
func fieldTypeName(f gen.Field) string {
	if f.Type.Type.Float() {
		if f.Type.Type == field.TypeFloat64 {
			return "Float"
		}
	} else if f.IsTime() {
		return "Time"
	}
	return f.Type.String()
}

func clearComment(f gen.Field) string {
	c := f.Comment()
	v := strings.ReplaceAll(c, "\\r", "")
	v = strings.ReplaceAll(v, "\\n", "")
	return v
}
