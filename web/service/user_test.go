package service

import (
	"testing"
	"x-ui/database"
	"x-ui/database/model"
	passwordutil "x-ui/util/password"
)

func TestCheckUserMigratesLegacyPasswordAndClearsPlaintext(t *testing.T) {
	initTestDB(t)

	db := database.GetDB()
	legacy := &model.User{
		Username: "legacy",
		Password: "legacy-pass",
	}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy user: %v", err)
	}

	service := &UserService{}
	user, ok := service.CheckUserCredentials("legacy", "legacy-pass")
	if !ok {
		t.Fatal("expected legacy credentials to validate")
	}
	if user.Password != "" {
		t.Fatalf("expected returned user password to be blank, got %q", user.Password)
	}

	reloaded := &model.User{}
	if err := db.First(reloaded, legacy.Id).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Password != "" {
		t.Fatalf("expected legacy plaintext to be cleared, got %q", reloaded.Password)
	}
	if reloaded.PasswordHash == "" {
		t.Fatal("expected password hash to be populated")
	}
	if !passwordutil.Verify(reloaded.PasswordHash, "legacy-pass") {
		t.Fatal("expected migrated hash to validate")
	}
}

func TestUpdateUserStoresHashOnly(t *testing.T) {
	initTestDB(t)

	service := &UserService{}
	user, err := service.GetFirstUser()
	if err != nil {
		t.Fatalf("get first user: %v", err)
	}

	if err := service.UpdateUser(user.Id, "updated-user", "updated-pass"); err != nil {
		t.Fatalf("update user: %v", err)
	}

	reloaded := &model.User{}
	if err := database.GetDB().First(reloaded, user.Id).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Username != "updated-user" {
		t.Fatalf("expected updated username, got %q", reloaded.Username)
	}
	if reloaded.Password != "" {
		t.Fatalf("expected blank plaintext password, got %q", reloaded.Password)
	}
	if !passwordutil.Verify(reloaded.PasswordHash, "updated-pass") {
		t.Fatal("expected password hash to validate new password")
	}
	if !service.ValidatePassword(user.Id, "updated-pass") {
		t.Fatal("expected ValidatePassword to accept current password")
	}
}
