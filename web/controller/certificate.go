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
	g.GET("/discover", a.discover)
	g.POST("/import-discovered", a.checkCSRF, a.importDiscovered)
}

func (a *CertificateController) list(c *gin.Context) {
	certificates, err := a.certificateService.List()
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, certificates, nil)
}

func (a *CertificateController) discover(c *gin.Context) {
	certificates, err := a.certificateService.Discover()
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, certificates, nil)
}

func (a *CertificateController) importDiscovered(c *gin.Context) {
	req := &service.ImportDiscoveredCertificateRequest{}
	if err := c.ShouldBind(req); err != nil {
		jsonObj(c, nil, err)
		return
	}
	certificate, err := a.certificateService.ImportDiscovered(req)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	jsonObj(c, certificate, nil)
}
