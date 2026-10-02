package db

import (
	"database/sql"
	"log"
	"os"
	"strings"
)

// Migration runner that creates Database's tables from /db/migrations sql files
func CreateDBTables(forumDB *sql.DB) error {
	log.Println("Running migrations...")

	// Read files from directory
	dirFiles, err := os.ReadDir("./internal/db/migrations")
	if err != nil {
		log.Fatalf("reading db/migrations folder files: %v", err)
	}

	// Collect only the .sql files
	for _, file := range dirFiles {
		filename := file.Name() // Take ONLY the string's name of the filename with method Name()
		if file.IsDir() || !strings.HasSuffix(filename, ".sql") {
			continue // skip the file if it IS a directory (method IsDir()) OR it is NOT am .sql file
		}

		path := "./internal/db/migrations/" + filename
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("sqlBytes file failed error: %v", err)
		}

		// Execute migration (.Exec() needs a string to execute)
		_, err = forumDB.Exec(string(sqlBytes)) // the content of each migration in ./db/migrations/
		if err != nil {
			log.Fatalf("execute migration: %s with error: %v", filename, err)
		}
	}

	log.Println("Migrations ran successfully")
	return nil
}
