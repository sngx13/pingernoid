package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func DBInit() {
	database, err := gorm.Open(
		sqlite.Open("pingernoid.db"),
		&gorm.Config{SkipDefaultTransaction: true},
	)
	if err != nil {
		log.Println("[!] 'DBInit' - Error, failed to connect to db...", err)
	}
	DB = database
}
