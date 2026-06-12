package session

import (
	"encoding/gob"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	loginUser = "LOGIN_USER"
)

type LoginUser struct {
	Id       int
	Username string
}

func init() {
	gob.Register(LoginUser{})
}

func SetLoginUser(c *gin.Context, user *LoginUser) error {
	s := sessions.Default(c)
	s.Set(loginUser, user)
	return s.Save()
}

func GetLoginUser(c *gin.Context) *LoginUser {
	s := sessions.Default(c)
	obj := s.Get(loginUser)
	if obj == nil {
		return nil
	}
	user := obj.(LoginUser)
	return &user
}

func IsLogin(c *gin.Context) bool {
	return GetLoginUser(c) != nil
}

func ClearSession(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Options(sessions.Options{
		Path:   "/",
		MaxAge: -1,
	})
	s.Save()
}
