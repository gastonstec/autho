// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package handles application routes
package routes

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger"
	"github.com/kueski-dev/paymentology-paymethods/handlers"
)

const CANNOT_SET_ROUTE = "%s: cannot set route"

// func InitRoutes initializes the application routes
func Set(app *fiber.App) error {
	var fr fiber.Router

	logger.LogInfo("Starting setting routes")


	// Probe route
	// Route that gets the service information
	fr = app.Get("/", handlers.Healthcheck)
	if fr == nil{
		return fmt.Errorf(CANNOT_SET_ROUTE, helpers.GetFunctionName())
	}

	// Route that gets the service information
	fr = app.Get("/authorizer/api/v1/admin/about", handlers.AdminAboutServiceHandler)
	if fr == nil{
		return fmt.Errorf(CANNOT_SET_ROUTE, helpers.GetFunctionName())
	}

	// Route to the Paymentology authorizer
	fr = app.Post("/authorizer/api/v1/pmtol/xmlrpc", handlers.AuthorizerXMLHandler)
	if fr == nil{
		return fmt.Errorf(CANNOT_SET_ROUTE, helpers.GetFunctionName())
	}

	logger.LogInfo("Routes has been set successfully")

	return nil
}
