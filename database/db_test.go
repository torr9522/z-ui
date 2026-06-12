package database

import (
	"path/filepath"
	"testing"
	"x-ui/database/model"
)

func TestInitDBBootstrapsSecureFirstUser(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "x-ui.db")

	bootstrap, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	if bootstrap == nil {
		t.Fatal("expected bootstrap credentials")
	}
	if bootstrap.Username == "" || bootstrap.Password == "" {
		t.Fatalf("expected non-empty bootstrap credentials: %#v", bootstrap)
	}
	if bootstrap.Username == "admin" || bootstrap.Password == "admin" {
		t.Fatalf("bootstrap credentials should not use defaults: %#v", bootstrap)
	}
	if bootstrap.WebPort < 10000 || bootstrap.WebPort > 59999 {
		t.Fatalf("unexpected bootstrap port: %d", bootstrap.WebPort)
	}

	user := &model.User{}
	if err := GetDB().First(user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Password != "" {
		t.Fatalf("expected blank legacy password column, got %q", user.Password)
	}
	if user.PasswordHash == "" {
		t.Fatal("expected password hash")
	}

	setting := &model.Setting{}
	if err := GetDB().Where("key = ?", "webPort").First(setting).Error; err != nil {
		t.Fatalf("load webPort setting: %v", err)
	}
	if setting.Value == "" {
		t.Fatal("expected persisted webPort")
	}

	secondBootstrap, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("re-init db: %v", err)
	}
	if secondBootstrap != nil {
		t.Fatalf("expected no bootstrap credentials on existing db, got %#v", secondBootstrap)
	}
}
