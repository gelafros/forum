package main

import (
	"database/sql"
	"log"

	"forum/internal/migrations"
	"forum/internal/server"

	_ "github.com/mattn/go-sqlite3" // "_" means load the "sqlite3" driver but don't use it directly, but from within "Database/sql" package
)

func main() {
	forumDB, err := sql.Open("sqlite3", "./db/forum.db") // Open/Create "forum.db" Database (object)
	if err != nil {
		log.Println("Error opening Database:", err)
		return
	}
	defer forumDB.Close() // Close Database for freeing system resources

	// Check if Database is connecting succesfully
	err = forumDB.Ping()
	if err != nil {
		log.Println("Error connecting to Database:", err)
		return
	}

	// Create schema_migrations (DB history) table
	err = migrations.RunSchema(forumDB)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Forum Database connected successfully")

	server.StartServer()
}
