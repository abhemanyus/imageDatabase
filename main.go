package main

import (
	"database/sql"
	"log"
)

func main() {
	db, err := sql.Open("sqlite3", "test.db")
	fatalErr(err)
	database, err := CreateDB(db)
	fatalErr(err)
	warnErr(database.CreateTag("sfw", "safe for work"))
	warnErr(database.CreateTag("nsfw", "NOT safe for work"))
}

func fatalErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func warnErr(err error) bool {
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}
