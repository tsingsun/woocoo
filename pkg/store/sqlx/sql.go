package sqlx

import (
	"database/sql"
	"github.com/tsingsun/woocoo/pkg/conf"
)

// NewSqlDB return a sql.DB from conf
func NewSqlDB(cfg *conf.Configuration) *sql.DB {
	db, err := sql.Open(cfg.String("driverName"), cfg.String("dsn"))
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
