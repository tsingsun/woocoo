package entimport

import "testing"

func Test_generateSchema(t *testing.T) {
	type args struct {
		dialect string
		dsn     string
		output  string
		tables  []string
		gengql  bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "mysql-user", args: args{
				dialect: "mysql",
				dsn:     "root@tcp(localhost:3306)/adminx?parseTime=true",
				output:  "./tmp",
				tables:  []string{"opm_user"},
				gengql:  true,
			},
		},
		{
			name: "clickhouse", args: args{
				dialect: "clickhouse",
				dsn:     "clickhouse://localhost:9000/adminx?debug=true&dial_timeout=5s",
				output:  "./tmp",
				tables:  []string{"opm_user"},
				gengql:  true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := generateSchema(tt.args.dialect, tt.args.dsn, tt.args.output, tt.args.tables, tt.args.gengql); (err != nil) != tt.wantErr {
				t.Errorf("generateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
