// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Handles load transactions.
package services

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger" 
	wallet "github.com/kueski-dev/paymentology-paymethods/models/wallet"
	commons "github.com/kueski-dev/paymentology-paymethods/services/commons"
)

// Handles a Load Auth request
func LoadAuth(c *fiber.Ctx) (*commons.RespSingleInt, error) {
	var err error

	// Parse body
	req := new(commons.Req)
	err = c.BodyParser(req)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// map request to JSON
	reqJS, checksumData, err := commons.MapReqToJSON(req)
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
	err = commons.ProtectReqValues(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// convert request to JSON
	jsonReq, err := commons.MapToJSON(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// post transaction in the wallet
	_, err = wallet.PostTransaction(reqJS.Reference, reqJS.RequestAmount, commons.TX_TYPE_LOAD_AUTH, commons.TX_OPER_INFO,
		fmt.Sprintf("%s | %s", commons.RESP_CODE[commons.RESP_CODE_APPROVED], reqJS.Narrative), string(jsonReq))
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// return response
	return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
}