package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3" // "_" means load the "sqlite3" driver but don't use it directly, but from within "database/sql" package
)

func main() {
	forumDB, err := sql.Open("sqlite3", "./db/forum.db")
	if err != nil {
		log.Println("Error opening Database:", err)
		return
	}
	defer forumDB.Close()

	err = forumDB.Ping()
	if err != nil {
		log.Println("Error connecting to database:", err)
		return
	}

	log.Println("Forum Database connected successfully")
}
