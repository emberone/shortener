package db

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func Migrate() error {
	driver, err := postgres.WithInstance(DB, &postgres.Config{})

	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:./migrations",
		"postgres", driver)

	fmt.Printf("err: %v\n", err)

	err = m.Up()

	fmt.Printf("err: %v\n", err)

	if err != nil {
		return err
	}

	return nil
}
