package main

import (
	"log"

	"forum/internal/db"
	"forum/internal/server"

	_ "github.com/mattn/go-sqlite3" // "_" means load the "sqlite3" driver but don't use it directly, but from within "Database/sql" package
)

func main() {
	// Create DB.db (The DataBase file)
	db.CreateDB()

	defer db.DB.Close() // Close Database for freeing system resources from sql.Open inside db.go

	// Create tables inside DB (Database)
	if err := db.CreateDBTables(db.DB); err != nil {
		log.Fatal("Migration error: ", err)
	}

	// Starting Server :8080
	server.StartServer()
}
