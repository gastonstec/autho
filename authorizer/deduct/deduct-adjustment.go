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
package deduct

import (
	"context"
	"encoding/json"
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

// Deduct Adjustment JSON struct
type DeductAdjReqJSON struct {
	MethodName 		string 					`json:"method-name"`
	TerminalId  	string					`json:"terminal-id"`
	Reference  		string					`json:"reference"`
	RequestAmount	string					`json:"request-amount"`
	Narrative  		string					`json:"narrative"`
	TxData        	*map[string]string 		`json:"tx-data"`
	ReferenceID  	string					`json:"reference-id"`
	ReferenceDate  	string					`json:"reference-date"`
	TxID  			string					`json:"tx-id"`
	TxDate  		string					`json:"tx-date"`
}

const DEDUCT_ADJUSTMENT_TX_TYPE = "DEADJ"


// Handles a Deduct Adjustment Request
func DeductAdjustmentHandler(c *fiber.Ctx) (*commons.RespSingleInt, error) {
	var err error

	// set a default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED)

	// Parse body
	req := new(Req)
	err = c.BodyParser(req)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		gojlogger.LogError(utils.GetFunctionName() + string(c.Body()))
		// return default response
		return resp, err
	}

	// Get values
	reqJS:= new(DeductAdjReqJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	reqJS.RequestAmount = req.TagParams.TagParam[2].TagValue.Value
	reqJS.Narrative = req.TagParams.TagParam[3].TagValue.Value
	txData:= req.TagParams.TagParam[4].TagValue.Value
	reqJS.ReferenceID = req.TagParams.TagParam[5].TagValue.Value
	reqJS.ReferenceDate = req.TagParams.TagParam[6].TagValue.Value
	reqJS.TxID = req.TagParams.TagParam[7].TagValue.Value
	reqJS.TxDate = req.TagParams.TagParam[8].TagValue.Value

	// checksum not verified because the transaction has 
	// to be processed in any case

	// decode KLV data
	reqJS.TxData, err = commons.DecodeKLV(txData)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		// return default response
		return resp, err
	}

	// process deduct reversal
	resp, err = deductAdjustment(reqJS)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
	}

	// check that the reversal was processed
	if resp.TagParams[0].Value != commons.RESP_CODE_APPROVED {
		// change response to approved
		resp.TagParams[0].Value = commons.RESP_CODE_APPROVED
		// log error
		gojlogger.LogError(utils.GetFunctionName() + string(c.Body()))
	}

	// return deduct response
	return resp, nil
}


// Process the deduct reversal operation (WITHDRAW OPERATION)
func deductAdjustment(req *DeductAdjReqJSON) (*commons.RespSingleInt, error) {
	
	// set a default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR)

	// check function parameters
	if req == nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", " invalid parameter req=nil")
	}
	if req.Reference == "" || req.RequestAmount == "" || len(req.RequestAmount) < 2 {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", " invalid reference or amount parameter")
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return resp, err
	}

	// set transaction isolation level
	var ct pgconn.CommandTag
	ct, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
	if err != nil || ct.String() != "SET" {
		tx.Rollback(ctx)
		return resp, err
	}

	// lock wallet table at row level
	ct, err = tx.Exec(ctx, "LOCK TABLE wallet IN ROW EXCLUSIVE MODE")
	if err != nil || ct.String() != "LOCK TABLE" {
		tx.Rollback(ctx)
		return resp, err
	}

	// lock wallet row
	var row pgx.Row
	var availableBal, blockedBal float64
	row = tx.QueryRow(ctx, "SELECT available_balance, blocked_balance FROM wallet WHERE wallet_id = $1 FOR UPDATE", req.Reference)
	err = row.Scan(&availableBal, &blockedBal)
	if err != nil {
		tx.Rollback(ctx)
		return resp, err
	}

	// convert amount to float
	var fAmount float64
	fAmount, err = strconv.ParseFloat(req.RequestAmount[0:(len(req.RequestAmount)-2)]+"."+
		req.RequestAmount[(len(req.RequestAmount)-2):(len(req.RequestAmount))], 64)
	if err != nil {
		tx.Rollback(ctx)
		return resp, err
	}

	// update wallet checking that available_balance stays in the same value
	ct, err = tx.Exec(ctx, 
		"UPDATE wallet SET available_balance = available_balance - $1, blocked_balance = blocked_balance + $1 WHERE wallet_id = $2 AND available_balance = $3",
		fAmount, req.Reference, availableBal)
	if err != nil || ct.String() != "UPDATE 1" {
		tx.Rollback(ctx)
		return resp, err
	}

	// convert request to JSON
	var jsonStr []byte
	jsonStr, err = json.Marshal(req)
	if err != nil {
		tx.Rollback(ctx)
		return resp, err
	}

	// insert wallet transaction
	txID := uuid.New().String()
	ct, err = tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, req.Reference, DEDUCT_ADJUSTMENT_TX_TYPE, commons.TX_OPER_WITHDRAW, fAmount, 
		commons.RESP_CODE[commons.RESP_CODE_APPROVED] + " - " + req.Narrative, string(jsonStr))
	if err != nil || ct.String() != "INSERT 0 1" {
		tx.Rollback(ctx)
		return resp, err
	}

	// commit database transaction
	err = tx.Commit(ctx)
	if err != nil {
		tx.Rollback(ctx)
		return resp, err
	}

	// set approved response
	resp = commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED)

	return resp, nil
}
