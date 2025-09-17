package di

import (
	bootserver "myproject/pkg/boot"
	"myproject/pkg/irrl"
	"myproject/pkg/util"

	services "myproject/pkg/client"
	"myproject/pkg/config"
	db "myproject/pkg/database"
)

func InitializeEvent(conf config.Config) (*bootserver.ServerHttp, error) {

	sqlDB, err := db.ConnectPGDB(conf)
	if err != nil {
		return nil, err // Return early if there's an error connecting to the database
	}

	utilInitiator := util.NewInitiator(sqlDB)
	irrlRepository := irrl.NewRepository(sqlDB, utilInitiator)

	myService := services.MyService{Config: conf}
	irrlService := irrl.NewService(irrlRepository, myService, utilInitiator)
	// admjwt := middleware.Adminjwt{Config: conf}
	admjwt := irrl.Adminjwt{Config: conf}
	irrlHandler := irrl.NewHandler(irrlService, myService, admjwt, conf, utilInitiator)
	serverHttp := bootserver.NewServerHttp(*irrlHandler)

	return serverHttp, nil
}
