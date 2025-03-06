package sqlx

import (
	native "database/sql"
	"database/sql/driver"
	"github.com/stretchr/testify/suite"
	"github.com/tsingsun/woocoo/pkg/conf"
	"os"
	"testing"
)

type testDriver struct {
	dsn string
}

func (w *testDriver) Open(name string) (driver.Conn, error) {
	w.dsn = name
	return nil, nil
}

type testSuite struct {
	suite.Suite
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (t *testSuite) SetupSuite() {
	native.Register("testDriver", &testDriver{})
}

func (t *testSuite) TestNoEncPwd() {
	config := `
store:
  testDriver:
    driverName: testDriver
    dsn: root:123456@tcp(127.0.0.1:3306)
    maxIdleConns: 10
    maxOpenConns: 100
    connMaxLifetime: 1m
`
	cfg := conf.NewFromBytes([]byte(config)).Load()
	type args struct {
		configuration *conf.Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "testDriver", args: args{configuration: cfg.Sub("store.testDriver")}, wantErr: false},
		{name: "mysql no import", args: args{configuration: func() *conf.Configuration {
			cfg.Parser().Set("store.testDriver.driverName", "mysql")
			return cfg
		}()}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func() {
			if tt.wantErr {
				t.Panics(func() {
					NewSqlDB(tt.args.configuration)
				})
			} else {
				db := NewSqlDB(tt.args.configuration)
				t.NoError(db.Ping())
				t.Equal(db.Driver().(*testDriver).dsn, "root:123456@tcp(127.0.0.1:3306)")
			}
		})
	}
}

func (t *testSuite) TestNewSqlDB_EncPwd() {
	config := `
testDriver:
  driverName: testDriver
  dsn: root:${password}@tcp(127.0.0.1:3306)/dbname
  maxIdleConns: 10
  maxOpenConns: 100
  connMaxLifetime: 1m
  encryption:
    password: NDL8hS0DJAAupzRH99b1mvbjqTzpBUYsqIY8YBBszGDxtQ==
`
	t.Run("aes-gcm", func() {
		t.Require().NoError(os.Setenv("DB_SECRET_KEY", "7d9f4e8b12c6a3f5e1b0d8c2a5f7e891"))
		cfg := conf.NewFromBytes([]byte(config)).Load()
		db := NewSqlDB(cfg.Sub("testDriver"))
		t.NoError(db.Ping())
		expectedPwd := "123456"
		t.Contains(db.Driver().(*testDriver).dsn, expectedPwd)
	})
	t.Run("miss method", func() {
		t.Require().NoError(os.Setenv("DB_SECRET_KEY", "7d9f4e8b12c6a3f5e1b0d8c2a5f7e891"))
		cfg := conf.NewFromBytes([]byte(config)).Load()
		cfg.Parser().Set("testDriver.encryption.method", "wrong")
		t.Panics(func() {
			NewSqlDB(cfg.Sub("testDriver"))
		})
	})
	t.Run("empty", func() {
		cfg := conf.NewFromBytes([]byte(config)).Load()
		cfg.Parser().Set("testDriver.encryption.password", "")
		db := NewSqlDB(cfg.Sub("testDriver"))
		t.NoError(db.Ping())
		t.Contains(db.Driver().(*testDriver).dsn, pwdTpl)
	})
	t.Run("miss key", func() {
		cfg := conf.NewFromBytes([]byte(config)).Load()
		t.Require().NoError(os.Setenv("DB_SECRET_KEY", ""))
		t.Panics(func() {
			NewSqlDB(cfg.Sub("testDriver"))
		})
	})
}

func (t *testSuite) TestAesGcmEncryptor() {
	t.Run("wrong key", func() {
		enc := aesEncryptor{}
		t.Require().NoError(os.Setenv("DB_SECRET_KEY", "wrongkey"))
		cs, err := enc.Encrypt("123456")
		t.Error(err)
		t.Empty(cs)
	})
	t.Run("empty", func() {
		t.Require().NoError(os.Setenv("DB_SECRET_KEY", ""))
		enc := aesEncryptor{}
		cs, err := enc.Decrypt("1")
		t.Error(err)
		t.Empty(cs)
	})
	t.Run("normal", func() {
		t.Require().NoError(os.Setenv("DB_SECRET_KEY", "7d9f4e8b12c6a3f5e1b0d8c2a5f7e891"))
		enc := aesEncryptor{}
		cs, _ := enc.Encrypt("123456")
		t.NotEmpty(cs)
	})
	t.Run("decrypt", func() {
		t.Require().NoError(os.Setenv("DB_SECRET_KEY", "7d9f4e8b12c6a3f5e1b0d8c2a5f7e891"))
		enc := aesEncryptor{}
		ed, _ := enc.Encrypt("123456")
		ds, _ := enc.Decrypt(ed)
		t.Equal("123456", ds)
	})
	t.Run("decrypt err", func() {
		t.Require().NoError(os.Setenv("DB_SECRET_KEY", "7d9f4e8b12c6a3f5e1b0d8c2a5f7e891"))
		enc := aesEncryptor{}
		_, err := enc.Decrypt("*")
		t.Error(err)
	})
}
