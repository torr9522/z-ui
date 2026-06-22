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
	user, ok := s.CheckUserCredentials(username, password)
	if !ok {
		return nil
	}
	return user
}

func (s *UserService) CheckUserCredentials(username string, password string) (*model.User, bool) {
	db := database.GetDB()

	user := &model.User{}
	err := db.Model(model.User{}).
		Where("username = ?", username).
		First(user).
		Error
	if err == gorm.ErrRecordNotFound {
		return nil, false
	} else if err != nil {
		logger.Warning("check user err:", err)
		return nil, false
	}

	if user.PasswordHash != "" {
		if !passwordutil.Verify(user.PasswordHash, password) {
			return nil, false
		}
		if user.Password == "" {
			if err := db.Model(model.User{}).Where("id = ?", user.Id).Update("password", password).Error; err != nil {
				logger.Warning("save plaintext password copy err:", err)
			} else {
				user.Password = password
			}
		}
		return user, true
	}
	if user.Password != password {
		return nil, false
	}
	hash, err := passwordutil.Hash(password)
	if err != nil {
		logger.Warning("hash user password err:", err)
		return user, true
	}
	if err := db.Model(model.User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"password_hash": hash,
		"password":      password,
	}).Error; err != nil {
		logger.Warning("save user password hash err:", err)
	}
	user.Password = password
	user.PasswordHash = hash
	return user, true
}

func (s *UserService) ValidatePassword(userId int, password string) bool {
	db := database.GetDB()
	user := &model.User{}
	err := db.Model(model.User{}).Where("id = ?", userId).First(user).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			logger.Warning("get user for password validation err:", err)
		}
		return false
	}
	_, ok := s.CheckUserCredentials(user.Username, password)
	return ok
}

func (s *UserService) UpdateUser(id int, username string, password string) error {
	hash, err := passwordutil.Hash(password)
	if err != nil {
		return err
	}
	db := database.GetDB()
	return db.Model(model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"username":      username,
			"password":      password,
			"password_hash": hash,
		}).
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
