package sqlx

import (
	"database/sql"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"strings"
)

const (
	aesEnvKey = "DB_SECRET_KEY"
	pwdTpl    = "${password}"
)

type encryptionConfig struct {
	Method   string `json:"method"`
	Password string `json:"password"`
}

// NewSqlDB create a sql.DB instance from config.
//
// Configuration example:
//
//		dbname:
//		  driverName: testDriver
//		  dsn: root:123456@tcp(127.0.0.1:3306)
//		  maxIdleConns: 10
//		  maxOpenConns: 100
//		  connMaxLifetime:
//
//	Encrypted password example:
//		dbname:
//		  driverName: testDriver
//		  dsn: root:${password}@tcp(127.0.0.1:3306)
//		  maxIdleConns: 10
//		  maxOpenConns: 100
//		  connMaxLifetime:
//	      encryption:
//	        password: U2FsdGVkX1+tlVEqk7q5J4HmwH0tZg
//	        method: aes-gcm
//
// if use encrypted password, you need pass the env "DB_SECRET_KEY" which used AES-GCM encrypted by default
func NewSqlDB(cfg *conf.Configuration) *sql.DB {
	dsn := cfg.String("dsn")
	if cfg.IsSet("encryption") {
		var err error
		dsn, err = processDSN(cfg)
		if err != nil {
			panic(err)
		}
	}
	db, err := sql.Open(cfg.String("driverName"), dsn)
	if err != nil {
		panic(err)
	}
	if cfg.IsSet("maxIdleConns") {
		db.SetMaxIdleConns(cfg.Int("maxIdleConns"))
	}
	if cfg.IsSet("maxOpenConns") {
		db.SetMaxOpenConns(cfg.Int("maxOpenConns"))
	}
	if cfg.IsSet("connMaxLifetime") {
		db.SetConnMaxLifetime(cfg.Duration("connMaxLifetime"))
	}

	return db
}

func processDSN(cfg *conf.Configuration) (string, error) {
	original := cfg.String("dsn")
	encConfig := encryptionConfig{
		Method: encryptorAES,
	}
	if err := cfg.Sub("encryption").Unmarshal(&encConfig); err != nil {
		return original, err
	}
	if encConfig.Password == "" {
		return original, nil
	}

	encryptor := encryptors[encConfig.Method]
	if encryptor == nil {
		return "", fmt.Errorf("encryptor %s not registered", encConfig.Method)
	}
	pwd, err := encryptor.Decrypt(encConfig.Password)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(original, pwdTpl, pwd), nil
}
