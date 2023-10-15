package entimport

import (
	"database/sql"
	"github.com/tsingsun/woocoo/cmd/woco/entimport/internal/driver"
	"io"
	"os"
	"strings"
	"testing"
)

const (
	ckDSN       = "clickhouse://localhost:9000/?debug=false&dial_timeout=5s"
	myDSN       = "root@tcp(localhost:3306)/test?parseTime=true"
	myCreateDSN = "root@tcp(localhost:3306)/?parseTime=true"
	testTmp     = "../../../test/tmp"
)

func init() {
	//createTestMysql()
	//createTestCK()
}

func createTestMysql() error {
	db, err := sql.Open("mysql", myCreateDSN)
	if err != nil {
		return err
	}
	fi, err := os.Open("testdata/mysql.sql")
	if err != nil {
		return err
	}
	bts, err := io.ReadAll(fi)
	if err != nil {
		return err
	}
	for _, s := range strings.Split(string(bts), ";") {
		if strings.TrimSpace(s) == "" {
			continue
		}
		_, err = db.Exec(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func createTestCK() error {
	db, err := sql.Open("clickhouse", ckDSN)
	if err != nil {
		return err
	}
	fi, err := os.Open("testdata/clickhouse.sql")
	if err != nil {
		return err
	}
	bts, err := io.ReadAll(fi)
	if err != nil {
		return err
	}
	for _, s := range strings.Split(string(bts), ";") {
		if strings.TrimSpace(s) == "" {
			continue
		}
		_, err = db.Exec(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func Test_generateSchema(t *testing.T) {
	type args struct {
		opts driver.ImportOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "mysql-user", args: args{
				opts: driver.ImportOptions{
					Dialect:       "mysql",
					DSN:           myDSN,
					SchemaPath:    testTmp,
					Tables:        []string{"entimport"},
					GenGraphql:    true,
					GenProtoField: true,
				},
			},
		},
		{
			name: "clickhouse", args: args{
				opts: driver.ImportOptions{
					Dialect:       "clickhouse",
					DSN:           ckDSN,
					SchemaPath:    testTmp,
					Tables:        []string{"entimport"},
					GenGraphql:    true,
					GenProtoField: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := generateSchema(tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("generateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
