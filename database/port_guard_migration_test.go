package database

import (
	"path/filepath"
	"testing"

	"x-ui/database/model"
)

func TestPortGuardMigrationAddsColumnsAndSettings(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "x-ui.db")
	if _, err := InitDB(dbPath); err != nil {
		t.Fatal(err)
	}

	columns := []string{
		"port_guard_enabled",
		"port_guard_window_seconds",
		"port_guard_ip_count",
		"port_guard_ban_seconds",
		"port_guard_banned_until",
		"port_guard_last_trigger_ip",
		"port_guard_last_trigger_at",
	}
	for _, column := range columns {
		ok, err := hasTableColumn(db, "inbounds", column)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("missing port guard column %s", column)
		}
	}

	for _, key := range []string{
		"portGuardEnabled",
		"portGuardWhitelistPorts",
		"portGuardSyncIntervalSeconds",
		"portGuardLogRetentionDays",
	} {
		var count int64
		if err := db.Model(&model.Setting{}).Where("key = ?", key).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("setting %s count = %d, want 1", key, count)
		}
	}
}
