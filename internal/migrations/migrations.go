package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Migration runner that create tables from /db/migrations sql files
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

		// Conversion prefix of filename eg. "001" to int 1
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

		} else {
			return errors.New("invalid version number lenth") // If version prefix in filename has length > 3
		}
	}
	return nil
}
