package config

import (
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/protocol"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	// handler
	h = &handler{}
)

type handler struct {
	service SyncConfigServiceServer
}

func (h *handler) Config() error {
	h.service = app.GetGrpcApp(appName).(SyncConfigServiceServer)
	return nil
}

func (h *handler) Name() string {
	return appName
}

func (h *handler) Registry(r *gin.Engine, subPath string) {
	userSubRouter := r.Group(protocol.ApiV1).Group(subPath)

	userSubRouter.POST("/sync", h.SyncConfigRouter)

}

// SyncConfigRouter
// @Summary Synchronization Profile Interface
// @Description in the Kubernetes cluster to obtain the contents of the ConfigMap, the use of confd landing generated into the corresponding software format configuration file
// @Tags synchronization Profile Interface
// @Accept application/json
// @Produce application/json
// @Param SyncConfigRequest body SyncConfigRequest true "ConfigMap information"
// @Success 200 {object} SyncConfigResponse
// @Failure 400 {object} SyncConfigResponse
// @Failure 500 {object} SyncConfigResponse
// @Router /config/sync [post]
func (h *handler) SyncConfigRouter(c *gin.Context) {
	req := &SyncConfigRequest{}

	if err := c.BindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, SyncConfigResponse{
			Message: fmt.Sprintf("Request body binding failed, error: %v", err),
		})
	} else {
		resp, err := h.service.SyncConfig(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, resp)
		} else {
			c.JSON(http.StatusCreated, resp)
		}
	}
}

func init() {
	app.RegistryHttpApp(h)
}
