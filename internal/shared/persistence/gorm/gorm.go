package gorm

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DbOptions struct {
	Host         string
	DbName       string
	User         string
	Password     string
	Schema       string
	MaxOpenConns int
	MaxIdleConns int
}

func NewDB(opts DbOptions) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		opts.Host, opts.User, opts.Password, opts.DbName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}

	if opts.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	}
	if opts.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("connected on pg")

	if opts.Schema != "" {
		if err := db.Exec(fmt.Sprintf("SET search_path TO %s", opts.Schema)).Error; err != nil {
			log.Fatal(err)
		}
	}

	return db
}
