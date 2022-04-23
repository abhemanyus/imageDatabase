package main

import (
	"database/sql"
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.Lshortfile)
	db, err := sql.Open("sqlite3", "test.db")
	fatalErr(err)
	database, err := CreateDB(db)
	fatalErr(err)
	server, err := CreateServer(database, "root")
	fatalErr(err)
	http.ListenAndServe(":8080", server)
}

func fatalErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
