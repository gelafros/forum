package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
)

// Migration runner that creates Database's tables from /db/migrations sql files
func CreateDBTables(forumDB *sql.DB) error {
	// Read files from directory
	dirFiles, err := os.ReadDir("./internal/db/migrations")
	if err != nil {
		log.Fatal(fmt.Printf("reading db/migrations folder files: %s", err))
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
			log.Fatal(fmt.Printf("sqlBytes file failed error: %s", err))
		}

		// Execute migration (.Exec() needs a string to execute)
		_, err = forumDB.Exec(string(sqlBytes)) // the content of each migration in ./db/migrations/
		if err != nil {
			log.Fatal(fmt.Printf("execute migration: %s with error: %s", filename, err))
		}
	}

	return nil
}
