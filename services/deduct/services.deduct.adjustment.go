// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package deduct handles the deduct transactions.
/* DeductAdjustment some business logic:
	The store of value must acknowledge this request for it to be handled appropriately. 
	cardholder has the right to dispute this transaction but the debit should happen first and
	before the dispute can be filed. If the store of value does not send an OK response to Tutuka,
	then the card will remain in a state of having a pending adjustment, and no transaction would
	be processed until the adjustment has been rectified.

	Adjustments must be accepted, even if there are not sufficient funds on the wallet. 
	You have to accept the adjustment and record the fact that you have this negative balance with the wallet, 
	regardless of whether or not you would actually ever show the wallet as having negative funds.
*/

// Handles deduct transactions.
package services

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger" 
	wallet "github.com/kueski-dev/paymentology-paymethods/models/wallet"
	commons "github.com/kueski-dev/paymentology-paymethods/services/commons"
)

// Handles a Deduct Adjustment request
func DeductAdjustment(c *fiber.Ctx) (*commons.RespSingleInt, error) {
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

	// get original deduct transaction
	originalTX, err := wallet.GetTransaction(reqJS.Reference, reqJS.ReferenceID, true)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}
	// check if original deduct transaction exists
	if originalTX == nil {
		// if original deduct tx not exists, it was never processed
		logger.LogWarning(helpers.GetFunctionName() + "- deduct adjustment without original deduct transaction with tx-id=" + reqJS.ReferenceID)
		return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
	}

	// convert request to JSON
	jsonReq, err := commons.MapToJSON(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// get wallet info
	walletInfo, err:= wallet.GetInfo(reqJS.Reference)
	if err != nil {
		return nil, err
	}

	// withdraw available balance
	err = wallet.WithdrawAvailableBalance(reqJS.Reference, reqJS.RequestAmount, walletInfo.AvalilableBalance, 
		commons.TX_TYPE_DEDUCT_ADJUSTMENT, 
		fmt.Sprintf("%s | original-tx-id=%s | %s", commons.RESP_CODE[commons.RESP_CODE_APPROVED], 
		reqJS.ReferenceID, reqJS.Narrative), string(jsonReq))
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}


	// return response
	return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
}