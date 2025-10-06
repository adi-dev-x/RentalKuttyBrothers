package bootserver

import (
	"fmt"
	"myproject/pkg/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type ServerHttp struct {
	engine *echo.Echo
}

// routeMounter is a tiny interface implemented by handlers that can mount routes
type routeMounter interface {
	MountRoutes(*echo.Echo)
}

// NewServerHttp accepts any number of handlers implementing MountRoutes and mounts them.
func NewServerHttp(handlers ...routeMounter) *ServerHttp {
	engine := echo.New()
	engine.Use(middleware.CORS())
	for _, h := range handlers {
		h.MountRoutes(engine)
	}
	return &ServerHttp{engine}
}

func (s *ServerHttp) Start(conf config.Config) {
	err := s.engine.Start(conf.Host + ":" + conf.ServerPort)
	if err != nil {
		fmt.Println("server error--", err.Error())
	}
}
