package service

import (
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/protocol"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	// handler service instance
	h = &handler{}
)

type handler struct {
	service ServiceLifecycleServer
}

func (h *handler) Config() error {
	h.service = app.GetGrpcApp(appName).(ServiceLifecycleServer)
	return nil
}

func (h *handler) Name() string {
	return appName
}

func (h *handler) Registry(r *gin.Engine, subPath string) {
	userSubRouter := r.Group(protocol.ApiV1).Group(subPath)

	userSubRouter.POST("/start", h.StartServiceRouter)
	userSubRouter.POST("/stop", h.StopServiceRouter)

}

// StartServiceRouter
// @Summary service Launch Interface
// @Description starting the service process in the container via supervisor
// @Tags service Launch Interface
// @Accept application/json
// @Produce application/json
// @Success 200 {object} ServiceResponse
// @Failure 400 {object} ServiceResponse
// @Failure 500 {object} ServiceResponse
// @Router /service/start [post]
func (h *handler) StartServiceRouter(c *gin.Context) {
	req := &ServiceRequest{}

	if resp, err := h.service.StartService(c, req); err != nil {
		c.JSON(http.StatusInternalServerError, resp)
	} else {
		c.JSON(http.StatusCreated, resp)
	}

}

// StopServiceRouter
// @Summary Service Discontinuation Interface
// @Description Stop the service process in the container via supervisor
// @Tags Service Discontinuation Interface
// @Accept application/json
// @Produce application/json
// @Success 200 {object} ServiceResponse
// @Failure 400 {object} ServiceResponse
// @Failure 500 {object} ServiceResponse
// @Router /service/stop [post]
func (h *handler) StopServiceRouter(c *gin.Context) {
	req := &ServiceRequest{}

	if resp, err := h.service.StopService(c, req); err != nil {
		c.JSON(http.StatusInternalServerError, resp)
	} else {
		c.JSON(http.StatusCreated, resp)
	}
}

func init() {
	app.RegistryHttpApp(h)
}
