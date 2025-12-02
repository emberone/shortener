package db

import (
	"context"
	"database/sql"
	"fmt"
	"shortener/internal/config"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

func Open(dsn string) *sql.DB {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		panic(err)
	}

	fmt.Printf("err: %v\n", err)

	return db
}

func Save() {
	sql.Drivers()
}

func Ping() bool {

	dbConn := Open(config.Server.DSN)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := dbConn.PingContext(ctx); err != nil {
		return true
	}

	return false
}
