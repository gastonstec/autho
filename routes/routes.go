package routes

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gastonstec/autho/utils"
	"github.com/gastonstec/autho/authorizer"
	"github.com/gastonstec/autho/routes/handlers"
)

const CANNOT_SET_ROUTE = "%s: cannot set route"

// func InitRoutes initializes the application routes
func InitRoutes(app *fiber.App) error {
	var fr fiber.Router

	// Route that gets the service information
	fr = app.Get("/authorizer/api/v1/admin/about", handlers.AdminAbout)
	if fr == nil{
		return fmt.Errorf(CANNOT_SET_ROUTE, utils.GetFunctionName())
	}

	// Route that get event(s)
	fr = app.Get("/authorizer/api/v1/auditlog/event/:event_id/:max_events", handlers.GetAuditEvents)
	if fr == nil{
		return fmt.Errorf(CANNOT_SET_ROUTE, utils.GetFunctionName())
	}

	// Route to the Paymentology authorizer
	fr = app.Post("/authorizer/api/v1/pmtol/xmlrpc", authorizer.XMLRPCRouter)
	if fr == nil{
		return fmt.Errorf(CANNOT_SET_ROUTE, utils.GetFunctionName())
	}

	return nil
}
