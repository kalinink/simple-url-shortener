package database

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // uses "file" to search for migration
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // register pq driver
	"time"
)

type Config struct {
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	MaxIdleConns    int
}

func Connect(connString string, cfg Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", connString)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	return db, nil
}

func MakeMigrations(db *sqlx.DB) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{MigrationsTable: "migration_version"})
	if err != nil {
		return err
	}
	migration, err := migrate.NewWithDatabaseInstance("file://internal/database/migrations", "hr_notifications", driver)
	if err != nil {
		return err
	}
	err = migration.Up()
	if err != nil {
		if err != migrate.ErrNoChange {
			return err
		}
	}
	return nil
}
