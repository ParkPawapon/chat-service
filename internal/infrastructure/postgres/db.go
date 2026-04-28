package postgres

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	gormDB *gorm.DB
	sqlDB  *sql.DB
}

func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	gormDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return &DB{gormDB: gormDB, sqlDB: sqlDB}, nil
}

func (db *DB) Gorm() *gorm.DB {
	return db.gormDB
}

func (db *DB) Close() error {
	if db == nil || db.sqlDB == nil {
		return nil
	}
	return db.sqlDB.Close()
}
