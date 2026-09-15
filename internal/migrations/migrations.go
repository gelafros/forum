package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Migration runner. RunSchema applies each new database migration and records it in schema_migrations.
func RunSchema(forumDB *sql.DB) error {
	// "schema_migrations" SQL command for creating the TABLE inside SQLite file (forum.db)
	schemaMigrationsSQLCmd := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);
		`

	// "updateSchemaMigrations" SQL command for UPDATING the schema_migrations with a change (append history)
	updateSchemaMigrations := `
		INSERT INTO schema_migrations (
			version, applied_at)
		VALUES (?,?);
	`

	// Execute the SQL cmd
	_, err := forumDB.Exec(schemaMigrationsSQLCmd) // Execute the SQL command for creating the TABLE "schema_migrations", inside forumDB database
	if err != nil {
		return fmt.Errorf("migration runner error: %w", err)
	}

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
			splittedFilename := strings.Split(filename, "_")
			version := splittedFilename[0]

			if len(version) == 3 {
				versionID, err := strconv.Atoi(version)
				if err != nil {
					return fmt.Errorf("conversion string to integer error: %w", err)
				}

				exists, err := checkSchemaMigrationsVer(forumDB, versionID)
				if err != nil {
					return fmt.Errorf("checking migrations version error: %w", err)
				}
				if exists {
					continue
				}

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

				_, err = forumDB.Exec(updateSchemaMigrations, versionID, time.Now().Format(time.RFC3339)) // run the cmd with parmeters (?,?) versionID and time.Now()
				if err != nil {
					return fmt.Errorf("updating schema_migrations TABLE for ID: %d with error: %w", versionID, err)
				}

			} else {
				return errors.New("invalid version number lenth") // If version prefix in filename has length > 3
			}
		}
	}

	return nil
}

// Check schema_migrations TABLE inside SQLite file (forum.db) for existing versions
func checkSchemaMigrationsVer(forumDB *sql.DB, versionID int) (bool, error) {
	var storedVersion int

	err := forumDB.QueryRow(
		"SELECT version FROM schema_migrations WHERE version = ?",
		versionID,
	).Scan(&storedVersion) // Scan checks whether the query returned a Row whose version equals versionID.

	if err == nil { // if the version from the Query HAS been found (existing migration)
		return true, nil
	}

	if err == sql.ErrNoRows { // if the version from the Query HAS NOT been found and it is a NEW one (new migration)
		return false, nil
	}

	return false, fmt.Errorf("schema_migrations error: %w", err) // Return error in every other case
}
