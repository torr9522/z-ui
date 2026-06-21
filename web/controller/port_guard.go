package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"x-ui/web/service"
	"x-ui/web/session"
)

type PortGuardController struct {
	BaseController

	portGuardService service.PortGuardService
}

func NewPortGuardController(g *gin.RouterGroup) *PortGuardController {
	a := &PortGuardController{}
	a.initRouter(g)
	return a
}

func (a *PortGuardController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/api/port-guard")
	g.Use(a.checkLogin)
	g.GET("/status", a.status)
	g.POST("/sync", a.checkCSRF, a.sync)
	g.POST("/unban", a.checkCSRF, a.unban)
	g.GET("/logs", a.logs)
}

func (a *PortGuardController) status(c *gin.Context) {
	status, err := a.portGuardService.Status()
	jsonObj(c, status, err)
}

func (a *PortGuardController) sync(c *gin.Context) {
	result, err := a.portGuardService.SyncNowBlocking()
	jsonObj(c, result, err)
}

func (a *PortGuardController) unban(c *gin.Context) {
	request := &service.PortGuardUnbanRequest{}
	if err := c.ShouldBind(request); err != nil {
		jsonMsg(c, "解除端口保护封禁", err)
		return
	}
	operator := "panel_user"
	if user := session.GetLoginUser(c); user != nil && user.Username != "" {
		operator = user.Username
	}
	err := a.portGuardService.Unban(request.Port, operator)
	jsonMsg(c, "解除端口保护封禁", err)
}

func (a *PortGuardController) logs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	logs, err := a.portGuardService.Logs(limit)
	jsonObj(c, logs, err)
}
