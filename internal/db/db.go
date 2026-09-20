package db

import (
	"database/sql"
	"log"
)

var ForumDB *sql.DB

// Creating or opening "forum.db" Database (object)
func CreateDB() error {

	var err error

	ForumDB, err = sql.Open("sqlite3", "./db/forum.db")
	if err != nil {
		log.Println("Error opening/creating the Database:", err)
		return err
	}

	// SQL foreign-key enforcement
	_, err = ForumDB.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Println("Error in activating foreign-key enforcement:", err)
		return err
	}

	// Check if Database is connecting succesfully
	err = ForumDB.Ping()
	if err != nil {
		log.Println("Error connecting to Database:", err)
		return err
	}

	return nil
}
