package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitDBSecuresDirectoryAndDatabasePermissions(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "etc", "x-ui")
	dbPath := filepath.Join(dbDir, "x-ui.db")

	if _, err := InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}

	dirInfo, err := os.Stat(dbDir)
	if err != nil {
		t.Fatalf("stat db dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("unexpected db dir permission: %#o", got)
	}

	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0600 {
		t.Fatalf("unexpected db file permission: %#o", got)
	}
}
