// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed
// Package deduct handles the deduct transactions.
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
package deduct

import (
	"context"
	"encoding/json"
	"strconv"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/gastonstec/autho/authorizer/commons"
	"github.com/gastonstec/autho/gojlogger"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/utils"
)

// Deduct Reversal JSON structs
type DeductRevReqJSON struct {
	MethodName 		string 					`json:"method-name"`
	TerminalId  	string					`json:"terminal-id"`
	Reference  		string					`json:"reference"`
	RequestAmount	string					`json:"request-amount"`
	Narrative  		string					`json:"narrative"`
	TxData        	string 					`json:"tx-data"`
	ReferenceID  	string					`json:"reference-id"`
	ReferenceDate  	string					`json:"reference-date"`
	TxID  			string					`json:"tx-id"`
	TxDate  		string					`json:"tx-date"`
}

const DEDUCT_REVERSAL_TX_TYPE = "DEREV"


// Handles a Deduct Reversal Request
func DeductReversalHandler(c *fiber.Ctx) (*commons.RespSingleInt, error) {
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
	reqJS := new(DeductRevReqJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	reqJS.RequestAmount = req.TagParams.TagParam[2].TagValue.Value
	reqJS.Narrative = req.TagParams.TagParam[3].TagValue.Value
	reqJS.TxData = req.TagParams.TagParam[4].TagValue.Value
	reqJS.ReferenceID = req.TagParams.TagParam[5].TagValue.Value
	reqJS.ReferenceDate = req.TagParams.TagParam[6].TagValue.Value
	reqJS.TxID = req.TagParams.TagParam[7].TagValue.Value
	reqJS.TxDate = req.TagParams.TagParam[8].TagValue.Value

	// checksum not verified because the transaction has 
	// to be processed in any case

	// KLV data not decoded because normally is empty

	// process deduct reversal
	resp, err = deductReversal(reqJS)
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

// Process the deduct reversal operation (DEPOSIT OPERATION)
func deductReversal(req *DeductRevReqJSON) (*commons.RespSingleInt, error) {
	var row pgx.Row
	var err error

	resp := commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR)

	// check function parameters
	if req == nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": invalid parameter req=nil")
	}
	if req.Reference == "" || req.RequestAmount == "" || len(req.RequestAmount) < 2 {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": invalid reference or amount parameter")
	}

	// get context
	ctx := context.Background()
	
	// get original deduct transaction
	row = db.DBRead.QueryRow(ctx, 
			"SELECT transaction_id FROM wallet_transaction WHERE wallet_id = $1 AND transaction_type_id = $2 AND transaction_data ->> 'tx-id' = $3", 
			req.Reference, DEDUCT_REVERSAL_TX_TYPE, req.ReferenceID)
	deductTxID := ""
	err = row.Scan(&deductTxID)
	if err != nil || deductTxID == "" {
		// if original deduct tx not exists, it was never processed
		gojlogger.LogWarning(utils.GetFunctionName() + " deduct reversal without original deduct transaction")
		return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
	}

	
	// begin database transaction
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

	// update wallet checking that available_balance stays in the same value
	ct, err= tx.Exec(ctx, 
		"UPDATE wallet SET available_balance = available_balance + $1, blocked_balance = blocked_balance - $1 WHERE wallet_id = $2 AND available_balance = $3",
		fAmount, req.Reference, availableBal)
	if err != nil || ct.String() != "UPDATE 1" {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}
		
	// convert request to JSON
	var jsonStr []byte
	jsonStr, err = json.Marshal(req)
	if err != nil {
		tx.Rollback(ctx)
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// insert wallet transaction
	txID := uuid.New().String()
	txOperation := commons.TX_OPER_DEPOSIT
	ct, err = tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, req.Reference, DEDUCT_REVERSAL_TX_TYPE, txOperation, fAmount, commons.RESP_CODE[commons.RESP_CODE_APPROVED] + " - " + req.Narrative, string(jsonStr))
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

	// set approved response
	resp = commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED)

	return resp, nil
}

