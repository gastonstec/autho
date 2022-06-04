package authorizer

import (
	"encoding/xml"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gastonstec/autho/gojlogger"
	"github.com/gastonstec/autho/memdb"
	"github.com/gastonstec/autho/authorizer/commons"
	"github.com/gastonstec/autho/authorizer/deduct"
	"github.com/gastonstec/autho/authorizer/load"
	"github.com/gastonstec/autho/authorizer/others"
)

const(
	RESPONSE_DO_NOT_HONOR = "<methodResponse><params><param><value><struct><member><name>resultCode</name><value><int>-9</int></value></member></struct></value></param></params></methodResponse>"
	RESPONSE_ZERO_BALANCE = "<methodResponse><params><param><value><struct><member><name>resultCode</name><value><int>1</int></value></member><member><name>balanceAmount</name><value><int>000</int></value></member></struct></value></param></params></methodResponse>"
	RESPONSE_INCORRECT_PIN = "<methodResponse><params><param><value><struct><member><name>resultCode</name><value><int>-25</int></value></member></struct></value></param></params></methodResponse>"
)

// Request struct
type Req struct {
	XMLName    xml.Name     `xml:"methodCall"`
	MethodName string       `xml:"methodName"`
	TagParams  ReqParamsTag `xml:"params"`
}
type ReqParamsTag struct {
	TagParam []ReqParam `xml:"param"`
}
type ReqParam struct {
	TagValue ValueStr `xml:"value"`
}
type ValueStr struct {
	Value string `xml:",any"`
}


// Single integer response struct
type RespSingleIntJSON struct {
	ResultCode string `json:"result-code"`
}
type RespSingleInt struct {
	XMLName   xml.Name               `xml:"methodResponse"`
	TagParams [1]RespSingleIntMember `xml:"params>param>value>struct>member"`
}
type RespSingleIntMember struct {
	Name  string `xml:"name"`
	Value string `xml:"value>int"`
}


// Fault response struct
type RespFault struct {
	XMLName xml.Name `xml:"methodResponse"`
	Message string   `xml:"fault>value>string"`
}


// XML Request router struct
type XMLReqRouter struct {
	MethodCall xml.Name `xml:"methodCall"`
	MethodName string   `xml:"methodName"`
}


// Performs the authorizer initial activities
func StartAuthorizer(app *fiber.App) error {
	// log starting
	gojlogger.LogInfo("Starting paymentology authorizer")

	// load in-memory database
	gojlogger.LogInfo("Loading authorizer in-memory database...")
	err := memdb.Load()
	if err != nil {
		return err
	}

	// success
	gojlogger.LogInfo("Paymentology authorizer has been started successfully")
	return nil
}


//  xmlrpc handler function
func XMLRPCRouter(c *fiber.Ctx) error {
	var err error
	var methResp *commons.RespSingleInt
	//var balResp *others.BalanceResponse

	// Set content type to XML
	c.Set("Content-type", "text/xml; charset=utf-8")

	// Get body contents
	xmlreq := new(XMLReqRouter)
	err = c.BodyParser(xmlreq)
	if err != nil {
		// Send fault response
		msg := fmt.Sprintf("<methodResponse><fault><value><string>%s</string></value></fault></methodResponse>", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(msg)
	}

	// Select handler to execute
	switch xmlreq.MethodName {
		case "Deduct": {
			// call handler
			methResp, err = deduct.DeductHandler(c)
		}
		case "DeductReversal": {
			// call handler
			methResp, err = deduct.DeductReversalHandler(c)
		}
		case "DeductAdjustment": {
			// call handler
			methResp, err = deduct.DeductAdjustmentHandler(c)
		}
		case "LoadAdjustment": {
			// call handler
			methResp, err = load.LoadAdjustmentHandler(c)
		}
		case "Stop": {

			// call handler
			methResp, err = others.StopCardHandler(c)

		}
		case "LoadAuth": {

			// call handler
			methResp, err = load.LoadAuthHandler(c)

		}
		case "Balance": {
			// log operation and return
			gojlogger.LogInfo(fmt.Sprintf("paymentology-call: methodName=%s", xmlreq.MethodName))
			return c.Status(fiber.StatusBadRequest).SendString(RESPONSE_ZERO_BALANCE)	
		}
		case "ValidatePIN": {

			// log operation and return
			gojlogger.LogInfo(fmt.Sprintf("paymentology-call: methodName=%s", xmlreq.MethodName))
			return c.Status(fiber.StatusBadRequest).SendString(RESPONSE_INCORRECT_PIN)	
		}
		case "AdministrativeMessage": {

			// log operation and return
			gojlogger.LogInfo(fmt.Sprintf("paymentology-call: methodName=%s", xmlreq.MethodName))
			return c.Status(fiber.StatusBadRequest).SendString(RESPONSE_DO_NOT_HONOR)	
		}
		default: {
			// Send default response
			gojlogger.LogError("paymentology-call: methodName=Default")
			return c.Status(fiber.StatusBadRequest).SendString(RESPONSE_DO_NOT_HONOR)
		}
	}

	// check for errors
	if err != nil {
		// log error
		gojlogger.LogError(fmt.Sprintf("paymentology-call: methodName=%s error=%s", xmlreq.MethodName, err.Error()))
		// Send fault response
		return c.Status(fiber.StatusBadRequest).SendString(RESPONSE_DO_NOT_HONOR)
	}

	// send response
	var resp []byte
	resp, err = xml.Marshal(methResp)
	if err != nil {
		// log error
		gojlogger.LogError(fmt.Sprintf("paymentology-call: methodName=%s error=%s", xmlreq.MethodName, err.Error()))
		// Send fault response
		return c.Status(fiber.StatusBadRequest).SendString(RESPONSE_DO_NOT_HONOR)
	}
	
	return c.Status(fiber.StatusOK).Send(resp)
}
