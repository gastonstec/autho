// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed
// Package load handles the load transactions.
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
package load

import(
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gastonstec/autho/authorizer/commons"
	"github.com/gastonstec/autho/gojlogger"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/utils"
)

// Deduct Reversal JSON structs
type LoadAdjReqJSON struct {
	MethodName 		string 				`json:"method-name"`
	TerminalId  	string				`json:"terminal-id"`
	Reference  		string				`json:"reference"`
	RequestAmount	string				`json:"request-amount"`
	Narrative  		string				`json:"narrative"`
	TxData  		*map[string]string 	`json:"tx-data"`
	ReferenceID  	string				`json:"reference-id"`
	ReferenceDate  	string				`json:"reference-date"`
	TxID  			string				`json:"tx-id"`
	TxDate  		string				`json:"tx-date"`
}


const LOAD_ADJUSTMENT_TX_TYPE = "LOADJ"


// Handles a LoadAdjustment Request
func LoadAdjustmentHandler(c *fiber.Ctx) (*commons.RespSingleInt, error) {
	var err error

	// set a default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED)

	// Parse body and check for errors
	req:= new(Req)
	err = c.BodyParser(req)
	if err != nil {
		return nil, err
	}

	// Get values
	reqJS:= new(LoadAdjReqJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	reqJS.RequestAmount = req.TagParams.TagParam[2].TagValue.Value
	reqJS.Narrative = req.TagParams.TagParam[3].TagValue.Value
	txData := req.TagParams.TagParam[4].TagValue.Value
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

	// process load adjustment
	resp, err = loadAdjustment(reqJS)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
	}


	// check that the adjustment was processed
	if resp.TagParams[0].Value != commons.RESP_CODE_APPROVED {
		// change response to approved
		resp.TagParams[0].Value = commons.RESP_CODE_APPROVED
		// log error
		gojlogger.LogError(utils.GetFunctionName() + string(c.Body()))
	}

	// return response
	return resp, nil
}


// Process a Load Adjustment
func loadAdjustment(req *LoadAdjReqJSON) (*commons.RespSingleInt, error) {
	var err error

	// set a default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR)

	// check function parameters
	if req == nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", " invalid parameter req=nil")
	}
	if req.Reference == "" || req.RequestAmount == "" || len(req.RequestAmount) < 2 {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", " invalid reference or amount parameter")
	}

	// convert amount to float
	var fAmount float64
	fAmount, err = strconv.ParseFloat(req.RequestAmount[0:(len(req.RequestAmount)-2)]+"."+
		req.RequestAmount[(len(req.RequestAmount)-2):(len(req.RequestAmount))], 64)
	if err != nil {
		return resp, err
	}
	
	// convert request to JSON
	var jsonStr []byte
	jsonStr, err = json.Marshal(req)
	if err != nil {
		return resp, err
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return resp, err
	}
	
	// insert wallet transaction
	txID := uuid.New().String()
	ct, err := tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, req.Reference, LOAD_ADJUSTMENT_TX_TYPE, commons.TX_OPER_INFO, fAmount, 
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

