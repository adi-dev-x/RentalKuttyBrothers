package main

import (
	"fmt"
	"os"

	"log"
	"myproject/pkg/config"
	db "myproject/pkg/database"
	"myproject/pkg/di"
)

func main() {
	// Load configuration
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Current working directory:", cwd)
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Point PG_HOST at the db-server IP stored in DynamoDB.
	if err := config.ResolveDBHost(&conf); err != nil {
		log.Fatal(err)
	}

	// Apply schema.sql and migration_product_code.sql before anything else
	// touches the database. Both files are idempotent, so this is safe to
	// run on every `go run`/`go build` startup.
	if err := db.RunSchemaMigrations(conf); err != nil {
		log.Fatalf("failed to apply database schema: %v", err)
	}

	server, err := di.InitializeEvent(conf)
	if err != nil {
		fmt.Printf("Error initializing Event: %s\n", err.Error())
		//log.Fatal("failed to initialize the files")
	}
	// fileServer := http.FileServer(http.Dir("./"))
	// server.Engine.GET("/*", echo.WrapHandler(fileServer))

	server.Start(conf)
}
