// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

/* LoadReversal some business logic:
	This is sent if we do not receive any response to the LoadAdjustment.
	It is an advice message which you must “approve”. 
	It will be repeated if you not respond to it.
	After 10 repeats, it will be logged for manual intervention.
	One big difference is that, almost without exception, a 
	LoadAdjustment will result in actual funds having been loaded to 
	a cardholder balance, and so reversing this means you will need 
	to “undo” that load. 
*/

// Package handles the load reversal transactions.
package services


import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger" 
	wallet "github.com/kueski-dev/paymentology-paymethods/models/wallet"
	commons "github.com/kueski-dev/paymentology-paymethods/services/commons"
)

// Handles a Load Reversal request
func LoadReversal(c *fiber.Ctx) (*commons.RespSingleInt, error) {
	var err error

	// Parse body
	req := new(commons.Req)
	err = c.BodyParser(req)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// map request to JSON
	reqJS, checksumData, err := commons.MapReqWithRefToJSON(req)
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
	err = commons.ProtectReqWithRefValues(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// get original transaction
	originalTX, err := wallet.GetTransaction(reqJS.Reference, reqJS.ReferenceID, true)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}
	// check if original transaction exists
	if originalTX == nil {
		// if original tx not exists, it was never processed
		logger.LogWarning(helpers.GetFunctionName() + "- load adjustment without original load adjustment transaction with tx-id=" + reqJS.ReferenceID)
		return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
	}

	// convert request to JSON
	jsonReq, err := commons.MapToJSON(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// withdraw available balance
	err = wallet.DepositBlockedBalance(reqJS.Reference, reqJS.RequestAmount, commons.TX_TYPE_LOAD_REVERSAL, 
		fmt.Sprintf("%s | original-tx-id=%s | %s", commons.RESP_CODE[commons.RESP_CODE_APPROVED], 
		reqJS.ReferenceID, reqJS.Narrative), string(jsonReq))
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// return response
	return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
}