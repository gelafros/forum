package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const migrationsDir = "./internal/db/migrations"

// CreateDBTables runs every .sql migration in internal/db/migrations in filename order.
func CreateDBTables(forumDB *sql.DB) error {
	log.Println("Running migrations...")

	dirFiles, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations folder: %w", err)
	}

	var filenames []string
	for _, file := range dirFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			filenames = append(filenames, file.Name())
		}
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, filename))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", filename, err)
		}
		if _, err := forumDB.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("executing migration %s: %w", filename, err)
		}
	}

	log.Println("Migrations ran successfully")
	return nil
}
