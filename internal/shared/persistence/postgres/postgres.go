package postgres

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type DbOptions struct {
	Host         string
	DbName       string
	User         string
	Password     string
	MaxOpenConns int
	MaxIdleConns int
}

func NewPostgres(opts DbOptions) *sql.DB {
	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=verify-full", opts.User, opts.Password, opts.Host, opts.DbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}

	fmt.Println("connected on pg")

	return db
}
