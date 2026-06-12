package database

import (
	"errors"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net"
	"os"
	"path"
	"x-ui/config"
	"x-ui/database/model"
	passwordutil "x-ui/util/password"
	"x-ui/util/random"
)

var db *gorm.DB

type BootstrapCredentials struct {
	Username string
	Password string
	WebPort  int
}

type legacyUser struct {
	Id       int `gorm:"primaryKey;autoIncrement"`
	Username string
	Password string
}

func (legacyUser) TableName() string {
	return "users"
}

func initUser() error {
	err := db.AutoMigrate(&legacyUser{})
	if err != nil {
		return err
	}
	return nil
}

func initInbound() error {
	return db.AutoMigrate(&model.Inbound{})
}

func initSetting() error {
	return db.AutoMigrate(&model.Setting{})
}

func InitDB(dbPath string) (*BootstrapCredentials, error) {
	dir := path.Dir(dbPath)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return nil, err
	}

	var gormLogger logger.Interface

	if config.IsDebug() {
		gormLogger = logger.Default
	} else {
		gormLogger = logger.Discard
	}

	c := &gorm.Config{
		Logger: gormLogger,
	}
	db, err = gorm.Open(sqlite.Open(dbPath), c)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(dbPath, 0600); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	err = initUser()
	if err != nil {
		return nil, err
	}
	err = initInbound()
	if err != nil {
		return nil, err
	}
	err = initSetting()
	if err != nil {
		return nil, err
	}
	err = runMigrations(dbPath)
	if err != nil {
		return nil, err
	}

	bootstrap, err := ensureBootstrapState()
	if err != nil {
		return nil, err
	}

	return bootstrap, nil
}

func GetDB() *gorm.DB {
	return db
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func ensureBootstrapState() (*BootstrapCredentials, error) {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil
	}

	username := fmt.Sprintf("admin_%s", random.SecureSeq(6))
	password := random.SecureSeq(24)
	passwordHash, err := passwordutil.Hash(password)
	if err != nil {
		return nil, err
	}

	webPort, err := ensureBootstrapPort()
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		PasswordHash: passwordHash,
	}
	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	return &BootstrapCredentials{
		Username: username,
		Password: password,
		WebPort:  webPort,
	}, nil
}

func ensureBootstrapPort() (int, error) {
	const (
		minPort = 10000
		maxPort = 60000
	)

	setting := &model.Setting{}
	err := db.Model(&model.Setting{}).Where("key = ?", "webPort").First(setting).Error
	if err == nil {
		return parsePort(setting.Value)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	port, err := pickBootstrapPort(minPort, maxPort)
	if err != nil {
		return 0, err
	}
	if err := db.Create(&model.Setting{Key: "webPort", Value: fmt.Sprintf("%d", port)}).Error; err != nil {
		return 0, err
	}
	return port, nil
}

func parsePort(raw string) (int, error) {
	var port int
	_, err := fmt.Sscanf(raw, "%d", &port)
	if err != nil {
		return 0, err
	}
	return port, nil
}

func pickBootstrapPort(minPort int, maxPort int) (int, error) {
	for attempts := 0; attempts < 64; attempts++ {
		port := random.SecureInt(minPort, maxPort)
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		_ = ln.Close()
		return port, nil
	}
	return 0, errors.New("unable to allocate bootstrap web port")
}
