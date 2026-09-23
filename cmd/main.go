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
	db.CreateDBTables(db.DB)

	log.Println("Forum Database connected successfully")

	// Starting Server :8080
	server.StartServer()
}
