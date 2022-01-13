package entimport

import (
	"testing"
)

func Test_action(t *testing.T) {
	dialect := "mysql"
	dsn := "root:123456@tcp(localhost:3306)/deo_account?parseTime=true"
	output := "./tmp"
	tables := []string{}
	err := generateSchema(dialect, dsn, output, tables)
	if err != nil {
		t.Fatal(err)
	}
}
