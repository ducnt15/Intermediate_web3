package database

import (
	"Intermediate_web3/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"os"
)

var db *bun.DB

func Connect() error {
	db = bun.NewDB(sql.OpenDB(
		pgdriver.NewConnector(pgdriver.WithDSN(os.Getenv("DSN")))), pgdialect.New())
	if db == nil {
		return fmt.Errorf("failed to initialize DB")
	}

	_, err := db.NewCreateTable().Model((*models.TrackingInformation)(nil)).IfNotExists().Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("error pinging the api: %w", err)
	}
	return nil
}

func Close() {
	if db != nil {
		err := db.Close()
		if err != nil {
			fmt.Printf("Error closing api: %v\n", err)
		}
	}
}

func GetDB() *bun.DB {
	return db
}
