package sql_test

import (
	native "database/sql"
	"database/sql/driver"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/store/sql"
	"testing"
)

type testDriver struct{}

func (w *testDriver) Open(name string) (driver.Conn, error) {
	return nil, nil
}

func TestNewBuiltInDB(t *testing.T) {
	{
		native.Register("testDriver", &testDriver{})
	}
	config := `
store:
  testDriver:
    driverName: testDriver
    dsn: root:123456@tcp(localhost:3306)
`
	cfg := conf.NewFromBytes([]byte(config)).Load()
	cfg.AsGlobal()
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "testDriver", args: args{path: "testDriver"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql.NewBuiltInDB(tt.args.path)
		})
	}
}
