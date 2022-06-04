// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package handlers contains the routes handlers
package handlers

import (
	"clevergo.tech/jsend"
	"github.com/gofiber/fiber/v2"
	"github.com/gastonstec/autho/auditlog"
	"strconv"
)

// Get service information
func AdminAbout(c *fiber.Ctx) error {

	// Create & fill data map
	data := make(map[string]string)
	data["service-name"] = "paymentology-paymethods"
	data["version"] = "1.00.00"
	data["appname"] = "Payment Methods Authorizer for Paymentology"

	
	// Send success response
	return c.Status(fiber.StatusOK).JSON(jsend.New(data))
}

// Get an event from the audit log
func GetAuditEvents(c *fiber.Ctx) error {
	var err error
	var max int = 100

	// Get parameters
	eventID := c.Params("event_id")
	maxEvents := c.Params("max_events")

	// check maxEvents parameter
	if maxEvents != "" {
		max, err = strconv.Atoi(maxEvents)
		if err != nil {
			max = 100
		}
	}

	// Get events
	var events []auditlog.LogEvent
	events, err = auditlog.GetEvents(eventID, max)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(jsend.New(err.Error()))
	}

	// success response
	return c.Status(fiber.StatusOK).JSON(jsend.New(&events))
}
