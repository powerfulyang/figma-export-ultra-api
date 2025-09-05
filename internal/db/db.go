package db

import (
	"database/sql"
	"log"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/internal/config"
)

var baseDB *sql.DB

// Open opens a DB connection using pgx and returns an Ent client.
func Open(cfg *config.Config) (*ent.Client, func(), error) {
	sqldb, err := sql.Open("pgx", cfg.PG.URL)
	if err != nil {
		return nil, func() {}, err
	}
	sqldb.SetMaxOpenConns(cfg.PG.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.PG.MaxIdleConns)
	baseDB = sqldb

	drv := entsql.OpenDB(dialect.Postgres, sqldb)
	client := ent.NewClient(ent.Driver(drv))
	closer := func() {
		baseDB = nil
		if err := client.Close(); err != nil {
			log.Printf("close ent client: %v", err)
		}
	}
	return client, closer, nil
}

// UpdatePool updates DB pool settings at runtime.
func UpdatePool(maxOpen, maxIdle int) {
	if baseDB == nil {
		return
	}
	if maxOpen > 0 {
		baseDB.SetMaxOpenConns(maxOpen)
	}
	if maxIdle >= 0 {
		baseDB.SetMaxIdleConns(maxIdle)
	}
}
