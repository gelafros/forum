package db

import (
	"database/sql"
	"log"
)

var DB *sql.DB

// Creating or opening "forum.db" Database (object)
func CreateDB() {
	var err error

	log.Println("Starting database...")
	DB, err = sql.Open("sqlite3", "./internal/db/forum.db?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		log.Fatal("Error opening/creating the Database:", err)
	}

	// Check if Database is connecting succesfully
	err = DB.Ping()
	if err != nil {
		log.Fatal("Error connecting to Database:", err)
	}

	log.Println("Database connected successfully")
}
