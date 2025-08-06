package protocol

import (
	"context"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"github.com/upmio/unit-operator/pkg/agent/docs"
	"net/http"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

const (
	ApiV1 = "/api/v1"
)

type HTTPService struct {
	r      *gin.Engine
	c      *conf.Config
	server *http.Server
}

func NewHTTPService() *HTTPService {
	c := conf.GetConf()

	if conf.GetConf().Log.GetLogLevel() != zap.DebugLevel {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(ginzap.Ginzap(zap.L().Named("[HTTP SERVICE]"), time.RFC3339, false))
	r.Use(ginzap.RecoveryWithZap(zap.L(), true))
	server := &http.Server{
		ReadHeaderTimeout: 60 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1M
		Addr:              fmt.Sprintf("%s:%d", c.App.Host, c.App.Port),
		Handler:           r,
	}

	return &HTTPService{
		server: server,
		c:      c,
		r:      r,
	}
}

func (s *HTTPService) EnableAPIRoot() {
	s.r.GET("/", s.apiRoot)
}

func (s *HTTPService) EnableSwagger() {
	docs.SwaggerInfo.BasePath = ApiV1
	s.r.GET("/swagger/*any", ginSwagger.DisablingWrapHandler(swaggerfiles.Handler, "SWAGGER"))
}

type RouteInfo struct {
	Method       string `json:"method"`
	FunctionName string `json:"function_name"`
	Path         string `json:"path"`
}

func transferRouteInfo(r gin.RouteInfo) *RouteInfo {
	return &RouteInfo{
		Method:       r.Method,
		FunctionName: r.Handler,
		Path:         r.Path,
	}
}

func (s *HTTPService) apiRoot(c *gin.Context) {
	routesInfo := make([]*RouteInfo, 0, 10)
	for _, value := range s.r.Routes() {
		routesInfo = append(routesInfo, transferRouteInfo(value))
	}
	c.JSON(200, routesInfo)
}

// Start start service
func (s *HTTPService) Start() error {
	// sub-service routing
	app.LoadHttpApp(s.r)

	s.EnableAPIRoot()

	s.EnableSwagger()

	zap.L().Named("[HTTP SERVICE]").Sugar().Infof("success start HTTP service, listen address: [%s]", s.c.App.Addr())

	if err := s.server.ListenAndServe(); err != nil {

		return fmt.Errorf("start service error, %s", err.Error())
	}
	return nil

}

// Stop stop server
func (s *HTTPService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// graceful shutdown
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown timeout, force exit")
	}
	return nil
}
