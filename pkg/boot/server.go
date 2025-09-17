package bootserver

import (
	"fmt"
	"myproject/pkg/config"
	"myproject/pkg/irrl"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type ServerHttp struct {
	engine *echo.Echo
}

func NewServerHttp(irrlHandler irrl.Handler) *ServerHttp {
	engine := echo.New()
	engine.Use(middleware.CORS())
	irrlHandler.MountRoutes(engine)
	//return &ServerHttp{Engine: engine}
	return &ServerHttp{engine}
}

func (s *ServerHttp) Start(conf config.Config) {
	err := s.engine.Start(conf.Host + ":" + conf.ServerPort)
	if err != nil {
		fmt.Println("server error--", err.Error())
	}
}
