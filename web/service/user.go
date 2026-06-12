package service

import (
	"errors"
	"gorm.io/gorm"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/logger"
	passwordutil "x-ui/util/password"
)

type UserService struct {
}

func (s *UserService) GetFirstUser() (*model.User, error) {
	db := database.GetDB()

	user := &model.User{}
	err := db.Model(model.User{}).
		First(user).
		Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) CheckUser(username string, password string) *model.User {
	db := database.GetDB()

	user := &model.User{}
	err := db.Model(model.User{}).
		Where("username = ?", username).
		First(user).
		Error
	if err == gorm.ErrRecordNotFound {
		return nil
	} else if err != nil {
		logger.Warning("check user err:", err)
		return nil
	}

	if user.PasswordHash != "" {
		if !passwordutil.Verify(user.PasswordHash, password) {
			return nil
		}
		user.Password = password
		return user
	}
	if user.Password != password {
		return nil
	}
	hash, err := passwordutil.Hash(password)
	if err != nil {
		logger.Warning("hash user password err:", err)
		return user
	}
	if err := db.Model(model.User{}).Where("id = ?", user.Id).Update("password_hash", hash).Error; err != nil {
		logger.Warning("save user password hash err:", err)
	}
	return user
}

func (s *UserService) UpdateUser(id int, username string, password string) error {
	hash, err := passwordutil.Hash(password)
	if err != nil {
		return err
	}
	db := database.GetDB()
	return db.Model(model.User{}).
		Where("id = ?", id).
		Update("username", username).
		Update("password", password).
		Update("password_hash", hash).
		Error
}

func (s *UserService) UpdateFirstUser(username string, password string) error {
	if username == "" {
		return errors.New("username can not be empty")
	} else if password == "" {
		return errors.New("password can not be empty")
	}
	db := database.GetDB()
	user := &model.User{}
	err := db.Model(model.User{}).First(user).Error
	if database.IsNotFound(err) {
		hash, err := passwordutil.Hash(password)
		if err != nil {
			return err
		}
		user.Username = username
		user.Password = password
		user.PasswordHash = hash
		return db.Model(model.User{}).Create(user).Error
	} else if err != nil {
		return err
	}
	hash, err := passwordutil.Hash(password)
	if err != nil {
		return err
	}
	user.Username = username
	user.Password = password
	user.PasswordHash = hash
	return db.Save(user).Error
}
