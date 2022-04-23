package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.Lshortfile)
	err := godotenv.Load(".env")
	fatalErr(err)
	db, err := sql.Open("sqlite3", os.Getenv("DATABASE"))
	fatalErr(err)
	database, err := CreateDB(db)
	fatalErr(err)
	server, err := CreateServer(database, os.Getenv("ROOT"))
	fatalErr(err)
	http.ListenAndServe(os.Getenv("ADDR"), server)
}

func fatalErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
