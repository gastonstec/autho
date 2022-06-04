// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed
// Package deduct handles the deduct transactions.
/* Deduct some business logic:
	When a cardholder makes an ATM, point of sale (POS), or e-commerce transaction, 
	Tutuka will send a Deduct request for the funds to be deducted from the store of value. 
	You’ll need to respond with Approved (1) for the transaction to be concluded successfully.
*/
package deduct

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/gastonstec/autho/authorizer/commons"
	"github.com/gastonstec/autho/gojlogger"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/utils"
)

// Deduct JSON structs
type DeductReqJSON struct {
	MethodName    string             `json:"method-name"`
	TerminalId    string             `json:"terminal-id"`
	Reference     string             `json:"reference"`
	RequestAmount string             `json:"request-amount"`
	Narrative     string             `json:"narrative"`
	TxType        string             `json:"tx-type"`
	TxData        *map[string]string `json:"tx-data"`
	TxID          string             `json:"tx-id"`
	TxDate        string             `json:"tx-date"`
}

type Req struct {
	XMLName    xml.Name     		`xml:"methodCall"`
	MethodName string       		`xml:"methodName"`
	TagParams  ReqParamsTag 		`xml:"params"`
}

type ReqParamsTag struct {
	TagParam []ReqParam 			`xml:"param"`
}

type ReqParam struct {
	TagValue ValueStr 				`xml:"value"`
}

type ValueStr struct {
	Value string 					`xml:",any"`
}

const DEDUCT_TX_TYPE = "DEDUC"


// Handles a Deduct Request
func DeductHandler(c *fiber.Ctx) (*commons.RespSingleInt, error) {
	var err error

	// set a default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR)

	// Parse body
	req := new(Req)
	err = c.BodyParser(req)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		// return default response
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// Get values
	reqJS := new(DeductReqJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	reqJS.RequestAmount = req.TagParams.TagParam[2].TagValue.Value
	reqJS.Narrative = req.TagParams.TagParam[3].TagValue.Value
	reqJS.TxType = req.TagParams.TagParam[4].TagValue.Value
	txData := req.TagParams.TagParam[5].TagValue.Value
	reqJS.TxID = req.TagParams.TagParam[6].TagValue.Value
	reqJS.TxDate = req.TagParams.TagParam[7].TagValue.Value

	// Validate checksum
	checksumData := reqJS.MethodName + reqJS.TerminalId + reqJS.Reference +
		reqJS.RequestAmount + reqJS.Narrative + reqJS.TxType +
		txData + reqJS.TxID + reqJS.TxDate

	checksum := req.TagParams.TagParam[8].TagValue.Value
	if commons.GetCheckSum(checksumData) != checksum {
		// log error
		gojlogger.LogError(utils.GetFunctionName() + " error=invalid checksum value")
		// send authentication fail response
		resp := commons.BuildSingleIntResp(commons.RESP_CODE_AUTHENTICATION_FAIL)
		return resp, nil
	}

	// decode KLV data
	reqJS.TxData, err = commons.DecodeKLV(txData)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		// return default response
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

		
	// process deduct
	resp, err = deduct(reqJS)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		return commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR), nil
	}

	// return deduct response
	return resp, nil
}


// Process a deduct operation (WITHDRAW OPERATION)
func deduct(req *DeductReqJSON) (*commons.RespSingleInt, error) {

	// set default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR)

	// check function parameters
	if req == nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + " invalid parameter req=nil")
	}
	if req.Reference == "" || req.RequestAmount == "" || len(req.RequestAmount) < 2 {
		return resp, fmt.Errorf(utils.GetFunctionName() + " invalid reference or amount parameter")
	}

	// check that the card is valid for operations
	card, err:= commons.GetCard(req.Reference, (*(*req).TxData)["LastfourDigitsPAN"])
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		// return default response
		return resp, nil
	}

	// card is not active
	if card == nil || card.StatusId != commons.CARD_STATUS_ACTIVE {
		gojlogger.LogInfo(fmt.Sprintf("%s card walletid=%s lastfour=%s is not active", utils.GetFunctionName(),
						req.Reference, (*(*req).TxData)["LastfourDigitsPAN"]))
		// return default response
		return commons.BuildSingleIntResp(commons.RESP_CODE_EXPIRED_CARD), nil
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// set transaction isolation level
	var ct pgconn.CommandTag
	ct, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
	if err != nil || ct.String() != "SET" {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// lock wallet table at row level
	ct, err = tx.Exec(ctx, "LOCK TABLE wallet IN ROW EXCLUSIVE MODE")
	if err != nil || ct.String() != "LOCK TABLE" {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// lock wallet row
	var row pgx.Row
	var availableBal, blockedBal float64
	row = tx.QueryRow(ctx, "SELECT available_balance, blocked_balance FROM wallet WHERE wallet_id = $1 FOR UPDATE", req.Reference)
	err = row.Scan(&availableBal, &blockedBal)
	if err != nil {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// convert amount to float
	var fAmount float64
	fAmount, err = strconv.ParseFloat(req.RequestAmount[0:(len(req.RequestAmount)-2)]+"."+
		req.RequestAmount[(len(req.RequestAmount)-2):(len(req.RequestAmount))], 64)
	if err != nil {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// set tx operation to withdraw by default
	txOperation := commons.TX_OPER_WITHDRAW

	// check available_balance vs amount
	if availableBal < fAmount {
		// set response to insufficient funds
		resp.TagParams[0].Value = commons.RESP_CODE_NOT_SUFF_FUNDS
		// set transaction operation to info
		txOperation = commons.TX_OPER_INFO
	}

	if txOperation == commons.TX_OPER_WITHDRAW {
		// update wallet checking that available_balance stays in the same value
		ct, err = tx.Exec(ctx, 
			"UPDATE wallet SET available_balance = available_balance - $1, blocked_balance = blocked_balance + $1 WHERE wallet_id = $2 AND available_balance = $3",
			fAmount, req.Reference, availableBal)
		if err != nil || ct.String() != "UPDATE 1" {
			tx.Rollback(ctx)
			return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
		}
	}

	// convert request to JSON
	var jsonStr []byte
	jsonStr, err = json.Marshal(req)
	if err != nil {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// insert wallet transaction
	// set approved response
	if resp.TagParams[0].Value == commons.RESP_CODE_DO_NOT_HONOR {
		resp.TagParams[0].Value = commons.RESP_CODE_APPROVED
	}
	txID := uuid.New().String()
	ct, err = tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, req.Reference, DEDUCT_TX_TYPE, txOperation, fAmount, 
		commons.RESP_CODE[resp.TagParams[0].Value] + " - " + req.Narrative, string(jsonStr))
	if err != nil || ct.String() != "INSERT 0 1" {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// commit database transaction
	err = tx.Commit(ctx)
	if err != nil {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	return resp, nil
}
