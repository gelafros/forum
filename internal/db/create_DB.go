package db

import (
	"database/sql"
	"log"
)

var DB *sql.DB

// Creating or opening "forum.db" Database (object)
func CreateDB() {
	var err error

	DB, err = sql.Open("sqlite3", "./internal/db/forum.db")
	if err != nil {
		log.Fatal("Error opening/creating the Database:", err)
	}

	// Check if Database is connecting succesfully
	err = DB.Ping()
	if err != nil {
		log.Fatal("Error connecting to Database:", err)
	}
	log.Println("Database connected successfully")

	// SQL foreign-key enforcement (for checking relationships between database tables.)
	_, err = DB.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal("Error in activating foreign-key enforcement:", err)
	}
}
