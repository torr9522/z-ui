package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"x-ui/util/random"
)

const csrfTokenKey = "CSRF_TOKEN"

func EnsureCSRFToken(c *gin.Context) (string, error) {
	s := sessions.Default(c)
	if token, ok := s.Get(csrfTokenKey).(string); ok && token != "" {
		return token, nil
	}
	token := random.SecureSeq(32)
	s.Set(csrfTokenKey, token)
	if err := s.Save(); err != nil {
		return "", err
	}
	return token, nil
}

func GetCSRFToken(c *gin.Context) string {
	s := sessions.Default(c)
	token, _ := s.Get(csrfTokenKey).(string)
	return token
}
