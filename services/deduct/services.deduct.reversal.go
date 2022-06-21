// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

/* DeductReversal some business logic:
	A reversal is essentially a request for a transaction that was not completed 
	and could have failed at a particular step of the transaction process.
	It is an advisement message to all parties of the transaction and ensures that 
	the card and store of value are put back into their original state if a 
	failed to deduct transaction had been initiated.
	Reversals are triggered by merchants for three reasons:
		1. The merchant did not receive any response back from the card scheme for the 
		authorization request. In this case, the merchant would timeout the transaction.
		2. The merchant receives a timeout request from the card scheme due to connectivity issues
		3. The merchant voided the transaction

	The store of value system will match the TransactionID of the initial Deduct and match it with
	the ReferenceID in the DeductReversal and Approves the DeductReversal by responding with 
	1 – Success and reverse the funds the funds back.
	The store of value system may not respond with a -9 Crashed or disapproved response.
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


// Handles a Deduct Request
func DeductReversal(c *fiber.Ctx) (*commons.RespSingleInt, error) {
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
		logger.LogWarning(helpers.GetFunctionName() + "- deduct reversal without original deduct transaction with tx-id=" + reqJS.ReferenceID)
		return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
	}

	// convert request to JSON
	jsonReq, err := commons.MapToJSON(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// withdraw blocked balance
	err = wallet.WithdrawBlockedBalance(reqJS.Reference, reqJS.RequestAmount, commons.TX_TYPE_DEDUCT_REVERSAL,
		fmt.Sprintf("%s | original-tx-id=%s | %s", commons.RESP_CODE[commons.RESP_CODE_APPROVED], reqJS.ReferenceID, reqJS.Narrative), string(jsonReq))
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// return response
	return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
}