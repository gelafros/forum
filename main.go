package main

import (
	"forum/internal/db"
	"forum/internal/migrations"
	"forum/internal/server"
	"log"

	_ "github.com/mattn/go-sqlite3" // "_" means load the "sqlite3" driver but don't use it directly, but from within "Database/sql" package
)

func main() {

	// Create forumDB.db (The DataBase file)
	err := db.CreateDB()
	if err != nil {
		log.Println(err)
		return
	}
	defer db.ForumDB.Close() // Close Database for freeing system resources from sql.Open inside db.go

	// Create tables inside ForumDB (Database)
	err = migrations.CreateDBTables(db.ForumDB)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Forum Database connected successfully")

	// Starting Server :8080
	server.StartServer()
}
