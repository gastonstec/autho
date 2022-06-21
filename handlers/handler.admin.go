// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package contains the admin routes handlers
package handlers

import (
	"clevergo.tech/jsend"
	"github.com/gofiber/fiber/v2"
)

// Health probe
func Healthcheck(c *fiber.Ctx) error {
	return c.SendStatus(200)
}

// Get service information
func AdminAboutServiceHandler(c *fiber.Ctx) error {

	// Create & fill data map
	data := make(map[string]string)
	data["service-name"] = "paymentology-paymethods"
	data["version"] = "1.00.00"
	data["appname"] = "Payment Methods Authorizer for Paymentology"

	
	// Send success response
	return c.Status(fiber.StatusOK).JSON(jsend.New(data))
}