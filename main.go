// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package main handles the application start.
package main

import (
	"log"
	"os"
	"os/signal"
	"github.com/gofiber/fiber/v2"
	"github.com/gastonstec/autho/utils"
	"github.com/gastonstec/autho/authorizer"
	"github.com/gastonstec/autho/gojlogger"
	"github.com/gastonstec/autho/config"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/routes"
)

const SERVICE_NAME = "paymentology-paymethods"

// Startup function
func main() {
	var err error

	// Start logger
	err = gojlogger.InitLogger("", SERVICE_NAME)
	if err != nil {
		log.Println("InitLogger error " + err.Error())
		os.Exit(1) // exit with error
	}

	// Load config variables
	err = config.LoadConfig()
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + ": " + err.Error())
		os.Exit(1) // exit with error
	}

	// open database connections
	err = db.OpenDB()
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + ": " + err.Error())
		os.Exit(1) // exit with error
	}
	defer db.CloseDB() // defer database closing

	// Create fiber application
	app := fiber.New()

	// Start paymethods/authorizer module
	err = authorizer.StartAuthorizer(app)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + ": " + err.Error())
		os.Exit(1) // exit with error
	}

	// Init routes
	gojlogger.LogInfo("Set routes...")
	err = routes.InitRoutes(app)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + ": " + err.Error())
		os.Exit(1) // exit with error
	}

	// log application environment
	gojlogger.LogInfo(utils.GetOsEnv())
	gojlogger.LogInfo(utils.GetGolangEnv())

	// Set shutdown application signal catch
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		sd := <-c
		gojlogger.LogInfo("Application shutdown started with " + sd.String() + " signal")
		err = app.Shutdown()
		gojlogger.LogInfo("Fiber application ended")
	}()

	// Start fiber server listening
	err = app.Listen(config.FiberPort)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + ": " + err.Error())
		os.Exit(1) // exit with error
	}

	// Cleanup tasks
	gojlogger.LogInfo("Final cleanup tasks")
}
