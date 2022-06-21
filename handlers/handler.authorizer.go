// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package contains the authorizer routes handlers
package handlers

import (
	"fmt"
	"encoding/xml"
	"github.com/gofiber/fiber/v2"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger"
	commons "github.com/kueski-dev/paymentology-paymethods/services/commons"
	deduct "github.com/kueski-dev/paymentology-paymethods/services/deduct"
	load "github.com/kueski-dev/paymentology-paymethods/services/load"
	others "github.com/kueski-dev/paymentology-paymethods/services/others"
)


// XML Request router struct
type XMLReqRouter struct {
	MethodCall xml.Name 	`xml:"methodCall"`
	MethodName string   	`xml:"methodName"`
}


const(
	RESPONSE_BODY_DO_NOT_HONOR = "<methodResponse><params><param><value><struct><member><name>resultCode</name><value><int>-9</int></value></member></struct></value></param></params></methodResponse>"
	RESPONSE_BODY_ZERO_BALANCE = "<methodResponse><params><param><value><struct><member><name>resultCode</name><value><int>1</int></value></member><member><name>balanceAmount</name><value><int>000</int></value></member></struct></value></param></params></methodResponse>"
	RESPONSE_BODY_INCORRECT_PIN = "<methodResponse><params><param><value><struct><member><name>resultCode</name><value><int>-25</int></value></member></struct></value></param></params></methodResponse>"

	RESPONSE_HEADER_USER_AGENT = "KueskiAuthorizer/1.0.0 (Go)"
	RESPONSE_HEADER_CONTENT_TYPE = "text/xml; charset=utf-8"

	REQUEST_BODY_MINIMUM_LENGTH = 50
	XMLROUTER_MSG_METHODNAME = "XMLRPCRouter methodName=%s"
)

//  xmlrpc handler function
func AuthorizerXMLHandler(c *fiber.Ctx) error {
	var err error
	var methResp *commons.RespSingleInt

	// check request body content
	if len(c.Body()) < REQUEST_BODY_MINIMUM_LENGTH {
		// Send fault response
		logger.LogError(helpers.GetFunctionName() + "- invalid request body content")
		return c.Status(fiber.StatusOK).SendString(RESPONSE_BODY_DO_NOT_HONOR)
	}

	// PRD DEBUGB
	logger.LogInfo(fmt.Sprintf("XMLRPCRouter RequestBody=%s", string(c.Body())))
	
	// parse body content
	xmlreq := new(XMLReqRouter)
	err = c.BodyParser(xmlreq)
	if err != nil {
		// Send fault response
		logger.LogError(fmt.Sprintf(helpers.GetFunctionName() + "- parsing body error=%s", err.Error()))
		return c.Status(fiber.StatusOK).SendString(RESPONSE_BODY_DO_NOT_HONOR)
	}

	// Set response headers
	c.Set("Content-type", RESPONSE_HEADER_CONTENT_TYPE)
	c.Set("User-Agent", RESPONSE_HEADER_USER_AGENT)

	// Select handler to execute
	switch xmlreq.MethodName {
		case "Deduct": {
			// call handler
			methResp, err = deduct.Deduct(c)
		}
		case "DeductReversal": {
			// call handler
			methResp, err = deduct.DeductReversal(c)
		}
		case "DeductAdjustment": {
			// call handler
			methResp, err = deduct.DeductAdjustment(c)
		}
		case "LoadAdjustment": {
			// call handler
			methResp, err = load.LoadAdjustment(c)
		}
		case "LoadReversal": {
			// call handler
			methResp, err = load.LoadReversal(c)
		}
		case "LoadAuth": {
			// call handler
			methResp, err = load.LoadAuth(c)
		}
		case "LoadAuthReversal": {
			// call handler
			methResp, err = load.LoadAuthReversal(c)
		}
		case "Stop": {
			// call handler
			methResp, err = others.StopCard(c)
		}
		case "Balance": {
			// log operation and return
			logger.LogInfo(fmt.Sprintf(XMLROUTER_MSG_METHODNAME, xmlreq.MethodName))
			return c.Status(fiber.StatusOK).SendString(RESPONSE_BODY_ZERO_BALANCE)	
		}
		case "ValidatePIN": {
			// log operation and return
			logger.LogInfo(fmt.Sprintf(XMLROUTER_MSG_METHODNAME, xmlreq.MethodName))
			return c.Status(fiber.StatusOK).SendString(RESPONSE_BODY_INCORRECT_PIN)	
		}
		case "AdministrativeMessage": {
			// log operation and return
			logger.LogInfo(fmt.Sprintf(XMLROUTER_MSG_METHODNAME, xmlreq.MethodName))
			return c.Status(fiber.StatusOK).SendString(RESPONSE_BODY_DO_NOT_HONOR)	
		}
		default: {
			// Send default response
			logger.LogError("XMLRPCRouter methodName=Default")
			return c.Status(fiber.StatusOK).SendString(RESPONSE_BODY_DO_NOT_HONOR)
		}
	}

	// check for errors
	if err != nil {
		// log error
		logger.LogError(fmt.Sprintf("XMLRPCRouter methodName=%s error=%s", xmlreq.MethodName, err.Error()))
		// Send fault response
		return c.Status(fiber.StatusOK).SendString(RESPONSE_BODY_DO_NOT_HONOR)
	}

	// send response
	var resp []byte
	resp, err = xml.Marshal(methResp)
	if err != nil {
		// log error
		logger.LogError(fmt.Sprintf("XMLRPCRouter methodName=%s error=%s", xmlreq.MethodName, err.Error()))
		// Send fault response
		return c.Status(fiber.StatusOK).SendString(RESPONSE_BODY_DO_NOT_HONOR)
	}

	// check response
	if resp == nil {
		// log error
		logger.LogError(fmt.Sprintf("XMLRPCRouter methodName=%s error=%s", xmlreq.MethodName, "response was nil, set to DO_NOT_HONOR"))
		resp = []byte(RESPONSE_BODY_DO_NOT_HONOR)
	}

	// PRD DEBUGB
	logger.LogInfo(fmt.Sprintf("XMLRPCRouter Response=%s", string(resp)))
	
	return c.Status(fiber.StatusOK).Send(resp)
}