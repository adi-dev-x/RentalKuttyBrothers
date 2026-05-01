package di

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	bootserver "myproject/pkg/boot"
	jobcron "myproject/pkg/cron"
	"myproject/pkg/irrl"
	"myproject/pkg/util"

	awscnf "github.com/aws/aws-sdk-go-v2/config"
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

	// Start nightly cron job: recalculates current_amount at 00:00 IST
	scheduler := jobcron.NewScheduler(irrlService)
	scheduler.Start()

	accessKey := conf.AWS_ACCKEY
	secretKey := conf.AWS_SECKEY
	awsCfg, err := awscnf.LoadDefaultConfig(context.TODO(),
		awscnf.WithRegion("ap-south-1"),
		awscnf.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to init AWS config: %w", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)
	admjwt := irrl.Adminjwt{Config: conf}
	irrlHandler := irrl.NewHandler(irrlService, myService, admjwt, conf, utilInitiator, s3Client)
	serverHttp := bootserver.NewServerHttp(*irrlHandler)

	return serverHttp, nil
}

