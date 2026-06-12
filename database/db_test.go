package database

import (
	"path/filepath"
	"strings"
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

func TestInitDBMigratesLegacyVLESSDecryption(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "x-ui.db")

	_, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	inbound := &model.Inbound{
		UserId:   1,
		Enable:   true,
		Port:     24002,
		Protocol: model.VLESS,
		Tag:      "inbound-24002",
		Settings: `{"clients":[{"id":"11111111-1111-1111-1111-111111111111"}]}`,
		Sniffing: `{}`,
	}
	if err := GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	if err := GetDB().Where("1 = 1").Delete(&model.SchemaMigration{}).Error; err != nil {
		t.Fatalf("clear migrations: %v", err)
	}

	_, err = InitDB(dbPath)
	if err != nil {
		t.Fatalf("re-init db: %v", err)
	}

	reloaded := &model.Inbound{}
	if err := GetDB().First(reloaded, inbound.Id).Error; err != nil {
		t.Fatalf("reload inbound: %v", err)
	}
	if !strings.Contains(reloaded.Settings, `"decryption":"none"`) {
		t.Fatalf("expected migrated decryption:none, got %s", reloaded.Settings)
	}
}
