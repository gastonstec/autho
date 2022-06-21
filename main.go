// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Handles the application start.
package main

import (
	"log"
	"os"
	"os/signal"
	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger"
	"github.com/kueski-dev/paymentology-paymethods/configs"
	"github.com/kueski-dev/paymentology-paymethods/db"
	"github.com/kueski-dev/paymentology-paymethods/services"
	"github.com/kueski-dev/paymentology-paymethods/routes"
)

const SERVICE_NAME = "paymentology-paymethods"
const OS_EXIT_CODE = 1 	// exit with error

// Startup function
func main() {
	var err error

	// start logger to os.Stdout
	err = logger.Start("", SERVICE_NAME)
	if err != nil {
		log.Println(err.Error())
		os.Exit(OS_EXIT_CODE)
	}

	// load config variables
	err = configs.LoadConfig()
	if err != nil {
		logger.LogError(helpers.GetFunctionName() + "- " + err.Error())
		os.Exit(OS_EXIT_CODE)
	}

	// log golang environment
	logger.LogInfo(helpers.GetGolangEnv())

	// open database connections
	err = db.OpenDB()
	if err != nil {
		logger.LogError(helpers.GetFunctionName() + "- " + err.Error())
		os.Exit(OS_EXIT_CODE)
	}
	defer db.CloseDB() // defer database closing

	// create fiber application
	app := fiber.New()
	// setup fiber logger
	app.Use(fiberlogger.New())

	// start services
	err = services.Start()
	if err != nil {
		logger.LogError(helpers.GetFunctionName() + "- " + err.Error())
		os.Exit(OS_EXIT_CODE)
	}

	// Init routes
	err = routes.Set(app)
	if err != nil {
		logger.LogError(helpers.GetFunctionName() + "- " + err.Error())
		os.Exit(OS_EXIT_CODE)
	}

	// set shutdown application signal catch
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		sd := <-c
		logger.LogInfo("Application shutdown started with " + sd.String() + " signal")
		err = app.Shutdown()
		logger.LogInfo("Fiber application ended")
	}()

	// start fiber server listening
	err = app.Listen(configs.FiberPort)
	if err != nil {
		logger.LogError(helpers.GetFunctionName() + "- " + err.Error())
		os.Exit(OS_EXIT_CODE)
	}

	// cleanup tasks
	logger.LogInfo("Perform final cleanup tasks")
}