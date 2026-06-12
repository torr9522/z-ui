package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
	"x-ui/database/model"
	"x-ui/util/password"
)

type migration struct {
	version string
	run     func(*gorm.DB) error
}

func runMigrations(dbPath string) error {
	migrations := []migration{
		{version: "202606120001_password_hash", run: migratePasswordHash},
		{version: "202606120002_inbound_clients", run: migrateInboundClients},
		{version: "202606120003_vless_decryption_none", run: migrateVLESSDecryption},
	}

	needed, err := needsMigration(migrations)
	if err != nil {
		return err
	}
	if needed {
		if err := backupDatabase(dbPath); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(&model.SchemaMigration{}); err != nil {
		return err
	}

	for _, item := range migrations {
		applied, err := isMigrationApplied(item.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := item.run(tx); err != nil {
				return err
			}
			return tx.Create(&model.SchemaMigration{Version: item.version}).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func needsMigration(migrations []migration) (bool, error) {
	exists, err := tableExists("schema_migrations")
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}

	for _, item := range migrations {
		applied, err := isMigrationApplied(item.version)
		if err != nil {
			return false, err
		}
		if !applied {
			return true, nil
		}
	}
	return false, nil
}

func tableExists(table string) (bool, error) {
	var name string
	err := db.Raw("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name).Error
	if err != nil {
		return false, err
	}
	return name == table, nil
}

func isMigrationApplied(version string) (bool, error) {
	var count int64
	err := db.Model(&model.SchemaMigration{}).Where("version = ?", version).Count(&count).Error
	return count > 0, err
}

func backupDatabase(dbPath string) error {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	backupPath := fmt.Sprintf("%s.bak.%s", dbPath, time.Now().UTC().Format("20060102150405.000000000"))
	src, err := os.Open(dbPath)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return err
	}
	dst, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func migratePasswordHash(tx *gorm.DB) error {
	hasColumn, err := hasTableColumn(tx, "users", "password_hash")
	if err != nil {
		return err
	}
	if !hasColumn {
		if err := tx.Exec("ALTER TABLE users ADD COLUMN password_hash TEXT").Error; err != nil {
			if !isDuplicateColumnError(err) {
				return err
			}
		}
	}

	var users []model.User
	if err := tx.Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if user.PasswordHash != "" || user.Password == "" {
			continue
		}
		hash, err := password.Hash(user.Password)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("password_hash", hash).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateInboundClients(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.InboundClient{}); err != nil {
		return err
	}

	var inbounds []model.Inbound
	if err := tx.Find(&inbounds).Error; err != nil {
		return err
	}
	for _, inbound := range inbounds {
		if err := migrateInboundSettingsClients(tx, &inbound); err != nil {
			return err
		}
	}
	return nil
}

func migrateVLESSDecryption(tx *gorm.DB) error {
	var inbounds []model.Inbound
	if err := tx.Where("protocol = ?", model.VLESS).Find(&inbounds).Error; err != nil {
		return err
	}
	for _, inbound := range inbounds {
		settings := map[string]interface{}{}
		if strings.TrimSpace(inbound.Settings) != "" {
			if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
				return err
			}
		}
		if value, ok := settings["decryption"].(string); ok && strings.TrimSpace(value) != "" {
			continue
		}
		settings["decryption"] = "none"
		data, err := json.Marshal(settings)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.Inbound{}).Where("id = ?", inbound.Id).Update("settings", string(data)).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateInboundSettingsClients(tx *gorm.DB, inbound *model.Inbound) error {
	settings := map[string]interface{}{}
	if strings.TrimSpace(inbound.Settings) != "" {
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			return err
		}
	}

	clients, ok := settings["clients"].([]interface{})
	if !ok || len(clients) == 0 {
		return nil
	}

	var existing int64
	if err := tx.Model(&model.InboundClient{}).Where("inbound_id = ?", inbound.Id).Count(&existing).Error; err != nil {
		return err
	}
	if existing == 0 {
		for _, client := range clients {
			clientSettings, err := json.Marshal(client)
			if err != nil {
				return err
			}
			record := &model.InboundClient{
				InboundId: inbound.Id,
				Email:     clientEmail(client),
				ClientKey: clientKey(client),
				Settings:  string(clientSettings),
				Enable:    true,
			}
			if err := tx.Create(record).Error; err != nil {
				return err
			}
		}
	}

	delete(settings, "clients")
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return tx.Model(&model.Inbound{}).Where("id = ?", inbound.Id).Update("settings", string(data)).Error
}

func hasTableColumn(tx *gorm.DB, table string, column string) (bool, error) {
	rows, err := tx.Raw("PRAGMA table_info(" + table + ")").Rows()
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func clientEmail(client interface{}) string {
	obj, ok := client.(map[string]interface{})
	if !ok {
		return ""
	}
	value, _ := obj["email"].(string)
	return value
}

func clientKey(client interface{}) string {
	obj, ok := client.(map[string]interface{})
	if !ok {
		return ""
	}
	for _, key := range []string{"id", "password", "email"} {
		value, _ := obj[key].(string)
		if value != "" {
			return value
		}
	}
	data, err := json.Marshal(client)
	if err != nil {
		return ""
	}
	return string(data)
}

func isDuplicateColumnError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate column name")
}
