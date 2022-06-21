// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

/* LoadAdjustment some business logic:
	As for a LoadAuth, this is the result of a refund or a money transfer request
	that loads funds to a cardholder but this is an advice message confirming that
	the funds have moved.
	When a LoadAuth is settled, it will result in a LoadAdjustment (this assumes 
	no LoadAuthReversal occurred). As with all advice messages, it is required that
	the message is approved. There is no other option except for a system failure/crash.
	Your response of “1” is not “approving” the adjustment, it is confirming that you 
	have been notified of the adjustment.
	All adjustments have already occurred and declining such a message will not change that. 
	If you respond with anything besides approval, it results in a process of manual intervention
	to investigate why the failure occurred and to work with you to take steps to correct it.
*/

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

// Handles a Load Adjustment request
func LoadAdjustment(c *fiber.Ctx) (*commons.RespSingleInt, error) {
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
		logger.LogWarning(helpers.GetFunctionName() + "- load adjustment without original deduct transaction with tx-id=" + reqJS.ReferenceID)
		return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
	}

	// convert request to JSON
	jsonReq, err := commons.MapToJSON(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// withdraw available balance
	err = wallet.WithdrawBlockedBalance(reqJS.Reference, reqJS.RequestAmount, commons.TX_TYPE_LOAD_ADJUSTMENT, 
		fmt.Sprintf("%s | original-tx-id=%s | %s", commons.RESP_CODE[commons.RESP_CODE_APPROVED], 
		reqJS.ReferenceID, reqJS.Narrative), string(jsonReq))
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// return response
	return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
}