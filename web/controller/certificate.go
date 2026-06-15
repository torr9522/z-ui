package controller

import (
	"github.com/gin-gonic/gin"
	"x-ui/web/service"
)

type CertificateController struct {
	BaseController

	certificateService service.CertificateService
}

func NewCertificateController(g *gin.RouterGroup) *CertificateController {
	a := &CertificateController{}
	a.initRouter(g)
	return a
}

func (a *CertificateController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/api/certificates")
	g.Use(a.checkLogin)
	g.GET("", a.list)
}

func (a *CertificateController) list(c *gin.Context) {
	certificates, err := a.certificateService.List()
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, certificates, nil)
}
