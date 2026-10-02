package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateDBTablesCreatesSchemaAndSeedCategories(t *testing.T) {
	// Verifies that all migrations create the required tables and seed categories.
	t.Chdir(filepath.Join("..", ".."))
	testDB, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "forum-test.db")+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	if err := CreateDBTables(testDB); err != nil {
		t.Fatal(err)
	}

	tables := []string{
		"users",
		"posts",
		"comments",
		"categories",
		"post_categories",
		"post_reactions",
		"comment_reactions",
		"sessions",
	}
	for _, table := range tables {
		var name string
		err := testDB.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q was not created: %v", table, err)
		}
	}

	var categoryCount int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categoryCount); err != nil {
		t.Fatal(err)
	}
	if categoryCount != 5 {
		t.Errorf("seeded category count = %d, want 5", categoryCount)
	}
}
