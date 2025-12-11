package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	//_ "github.com/mattn/go-sqlite3"
	//_ "modernc.org/sqlite"
)

var DB *sql.DB

func Open(driver, dsn string) error {

	var err error

	DB, err = sql.Open(driver, dsn)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err = DB.PingContext(ctx); err != nil {
		fmt.Printf("err: %v\n", err)
		panic(err)
	}

	return err

}

func Save() {
	sql.Drivers()
}

func Ping() bool {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := DB.PingContext(ctx); err == nil {
		return true
	}

	return false
}
