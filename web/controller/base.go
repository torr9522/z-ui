package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"x-ui/web/session"
)

type BaseController struct {
}

func (a *BaseController) checkLogin(c *gin.Context) {
	if !session.IsLogin(c) {
		if isAjax(c) {
			pureJsonMsg(c, false, "登录时效已过，请重新登录")
		} else {
			c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path"))
		}
		c.Abort()
	} else {
		c.Next()
	}
}

func (a *BaseController) checkCSRF(c *gin.Context) {
	token := session.GetCSRFToken(c)
	if token == "" {
		jsonMsg(c, "请求校验", errors.New("csrf token is missing"))
		c.Abort()
		return
	}
	if c.GetHeader("X-CSRF-Token") != token {
		jsonMsg(c, "请求校验", errors.New("csrf token is invalid"))
		c.Abort()
		return
	}
	c.Next()
}
