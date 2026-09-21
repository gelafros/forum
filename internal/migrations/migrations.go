package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// Migration runner that creates Database's tables from /db/migrations sql files
func CreateDBTables(forumDB *sql.DB) error {

	// Read files from directory
	dirFiles, err := os.ReadDir("./db/migrations")
	if err != nil {
		return fmt.Errorf("reading db/migrations folder files: %w", err)
	}

	for _, file := range dirFiles {
		filename := file.Name() // Take ONLY the string's name of the filename with method Name()
		if file.IsDir() {
			continue // skip the file if it IS a directory (method IsDir())
		}

		// Collect only the .sql files
		if strings.HasSuffix(filename, ".sql") {

			path := "./db/migrations/" + filename
			sqlBytes, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("sqlBytes file failed error: %w", err)
			}

			// Execute migration (.Exec() needs a string to execute)
			_, err = forumDB.Exec(string(sqlBytes)) // the content of each migration in ./db/migrations/
			if err != nil {
				return fmt.Errorf("execute migration: %s with error: %w ", filename, err)
			}
		}
	}
	return nil
}
