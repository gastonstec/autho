// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

package services

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger" 
	card "github.com/kueski-dev/paymentology-paymethods/models/card"
	commons "github.com/kueski-dev/paymentology-paymethods/services/commons"
)

// Handles a Stop Card request
func StopCard(c *fiber.Ctx) (*commons.RespSingleInt, error) {
	var err error

	// Parse body
	req := new(commons.Req)
	err = c.BodyParser(req)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// map request to JSON
	reqJS, checksumData, err := commons.MapStopReqToJSON(req)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// verify checksum
	if commons.GetCheckSum(checksumData) != reqJS.Checksum {
		logger.LogWarning(helpers.GetFunctionName() + 
				fmt.Sprintf(" authentication fail method=%s tx-id=%s", reqJS.MethodName, reqJS.TxID))
		return commons.BuildSingleIntResp(commons.RESP_CODE_AUTHENTICATION_FAIL), nil
	}
	// protect request values
	reqJS.TerminalId, reqJS.Checksum = "", ""
	
	// convert request to JSON
	jsonReq, err := commons.MapToJSON(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// post transaction in the wallet
	last4 := reqJS.VoucherNumber[len(reqJS.VoucherNumber)-4:len(reqJS.VoucherNumber)]
	err = card.Stop(reqJS.Reference, last4, 
		fmt.Sprintf("%s | CARD HAS BEEN STOPPED REASON_CODE=%s", commons.RESP_CODE[commons.RESP_CODE_APPROVED], reqJS.StopReason), 
		string(jsonReq))
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// return response
	return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
}