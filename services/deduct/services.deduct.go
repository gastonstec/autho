// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

/* Deduct some business logic:
	When a cardholder makes an ATM, point of sale (POS), or e-commerce transaction, 
	Tutuka will send a Deduct request for the funds to be deducted from the store of value. 
	You’ll need to respond with Approved (1) for the transaction to be concluded successfully.
*/

// Handles deduct transactions.
package services

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger"
	card "github.com/kueski-dev/paymentology-paymethods/models/card" 
	wallet "github.com/kueski-dev/paymentology-paymethods/models/wallet"
	commons "github.com/kueski-dev/paymentology-paymethods/services/commons"
)


// Handles a Deduct Request
func Deduct(c *fiber.Ctx) (*commons.RespSingleInt, error) {
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

	// get card info
	cardInfo, err:= card.GetInfo(reqJS.Reference, (*(*reqJS).TxData)["LastfourDigitsPAN"])
	if err != nil {
		logger.LogError(err.Error())
		return commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR), nil
	}

	// check card is active or expired
	if cardInfo == nil || !card.IsActive(cardInfo) {
		logger.LogInfo(fmt.Sprintf("%s - card walletid=%s lastfour=%s is not active", helpers.GetFunctionName(),
						reqJS.Reference, (*(*reqJS).TxData)["LastfourDigitsPAN"]))
		return commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR), nil
	}

	// get wallet info
	walletInfo, err:= wallet.GetInfo(reqJS.Reference)
	if err != nil {
		logger.LogError(err.Error())
		return commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR), nil
	}

	// check wallet is active
	if walletInfo == nil || !wallet.IsActive(walletInfo) {
		logger.LogInfo(fmt.Sprintf("%s - walletid=%s is not active", helpers.GetFunctionName(), reqJS.Reference))
		return commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR), nil
	}

	// convert request to JSON
	jsonReq, err := commons.MapToJSON(reqJS)
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// check for funds
	if walletInfo.AvalilableBalance <= reqJS.RequestAmount {
		wallet.PostTransaction(walletInfo.WalletId, reqJS.RequestAmount, commons.TX_TYPE_DEDUCT, commons.TX_OPER_INFO, 
					fmt.Sprintf("%s | %s", commons.RESP_CODE[commons.RESP_CODE_NOT_SUFF_FUNDS] , reqJS.Narrative), string(jsonReq))
		return commons.BuildSingleIntResp(commons.RESP_CODE_NOT_SUFF_FUNDS), nil
	}

	// withdraw available balance
	err = wallet.WithdrawAvailableBalance(reqJS.Reference, reqJS.RequestAmount, walletInfo.AvalilableBalance, 
					commons.TX_TYPE_DEDUCT, fmt.Sprintf("%s | %s", commons.RESP_CODE[commons.RESP_CODE_APPROVED], 
					reqJS.Narrative), string(jsonReq))
	if err != nil {
		return nil, commons.RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// return response
	return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
}