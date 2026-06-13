package service

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"x-ui/database"
	"x-ui/database/model"
)

func initTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "x-ui.db")
	if _, err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return dbPath
}

func TestInboundServicePersistsClientsSeparately(t *testing.T) {
	initTestDB(t)

	service := &InboundService{}
	inbound := &model.Inbound{
		UserId:   1,
		Enable:   true,
		Port:     24001,
		Protocol: model.VLESS,
		Tag:      "inbound-24001",
		Settings: `{"clients":[{"id":"11111111-1111-1111-1111-111111111111","email":"a@example.com","flow":"xtls-rprx-vision"}],"decryption":"none"}`,
		Sniffing: `{}`,
	}
	if err := service.AddInbound(inbound); err != nil {
		t.Fatalf("add inbound: %v", err)
	}

	saved, err := service.GetInbound(inbound.Id)
	if err != nil {
		t.Fatalf("get inbound: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(saved.Settings), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	clients, ok := settings["clients"].([]interface{})
	if !ok || len(clients) != 1 {
		t.Fatalf("expected reattached clients, got %#v", settings["clients"])
	}

	db := database.GetDB()
	var count int64
	if err := db.Model(&model.InboundClient{}).Where("inbound_id = ?", inbound.Id).Count(&count).Error; err != nil {
		t.Fatalf("count clients: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 inbound client, got %d", count)
	}
}

func TestInboundServiceNormalizesTrojanFlowOnSave(t *testing.T) {
	initTestDB(t)

	service := &InboundService{}
	inbound := &model.Inbound{
		UserId:   1,
		Enable:   true,
		Port:     24002,
		Protocol: model.Trojan,
		Tag:      "inbound-24002",
		Settings: `{"clients":[{"password":"secret-pass","flow":"xtls-rprx-vision"}]}`,
		Sniffing: `{}`,
	}
	if err := service.AddInbound(inbound); err != nil {
		t.Fatalf("add inbound: %v", err)
	}

	saved, err := service.GetInbound(inbound.Id)
	if err != nil {
		t.Fatalf("get inbound: %v", err)
	}
	if strings.Contains(saved.Settings, `"flow"`) {
		t.Fatalf("expected trojan flow to be removed from settings, got %s", saved.Settings)
	}
}

func TestInboundServiceNormalizesVMessAlterIDOnSave(t *testing.T) {
	initTestDB(t)

	service := &InboundService{}
	inbound := &model.Inbound{
		UserId:   1,
		Enable:   true,
		Port:     24003,
		Protocol: model.VMess,
		Tag:      "inbound-24003",
		Settings: `{"clients":[{"id":"11111111-1111-1111-1111-111111111111","alterId":64}]}`,
		Sniffing: `{}`,
	}
	if err := service.AddInbound(inbound); err != nil {
		t.Fatalf("add inbound: %v", err)
	}

	saved, err := service.GetInbound(inbound.Id)
	if err != nil {
		t.Fatalf("get inbound: %v", err)
	}
	if !strings.Contains(saved.Settings, `"alterId":0`) {
		t.Fatalf("expected vmess alterId to normalize to 0, got %s", saved.Settings)
	}
}

func TestInitDBCreatesBackupOnMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "x-ui.db")

	cmd := exec.Command("sqlite3", dbPath, "CREATE TABLE users (id integer primary key autoincrement, username text, password text);")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("seed sqlite db: %v (%s)", err, string(out))
	}
	if _, err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}

	matches, err := filepath.Glob(dbPath + ".bak.*")
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected migration backup for %s", dbPath)
	}
}
