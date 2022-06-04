package others

import(
	"fmt"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
	// "github.com/jackc/pgconn"
	// "github.com/jackc/pgx/v4"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/gojlogger"
	"github.com/gastonstec/autho/utils"
	"github.com/gastonstec/autho/authorizer/commons"
)

// Deduct JSON structs
type StopReqJSON struct {
	MethodName 		string 				`json:"method-name"`
	TerminalId  	string				`json:"terminal-id"`
	Reference  		string				`json:"reference"`
	VoucherNumber 	string				`json:"voucher-number"`
	StopReason 		string				`json:"stop-reason"`
	TxData        	*map[string]string 	`json:"tx-data"`
	TxID  			string				`json:"tx-id"`
	TxDate  		string				`json:"tx-date"`
}

const STOP_TX_TYPE = "CRDST"


// Handles a stop card request
func StopCardHandler(c *fiber.Ctx) (*commons.RespSingleInt, error) {
	var err error

	// set a default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR)

	// Parse body and check for errors
	req:= new(Req)
	err = c.BodyParser(req)
	if err != nil {
		return nil, err
	}

	// Get values
	reqJS:= new(StopReqJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	reqJS.VoucherNumber = req.TagParams.TagParam[2].TagValue.Value
	reqJS.StopReason = req.TagParams.TagParam[3].TagValue.Value
	txData := req.TagParams.TagParam[4].TagValue.Value
	reqJS.TxID = req.TagParams.TagParam[5].TagValue.Value
	reqJS.TxDate = req.TagParams.TagParam[6].TagValue.Value

	// Buid checksum data
	checksumData:= reqJS.MethodName + reqJS.TerminalId + reqJS.Reference +
				reqJS.VoucherNumber + reqJS.StopReason + txData +
				reqJS.TxID + reqJS.TxDate

	// Validate checksum
	checksum := req.TagParams.TagParam[7].TagValue.Value
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

	// stop card
	resp, err = stopCard(reqJS)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		return commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR), nil
	}

	return resp, nil
}


// Process a stop card
func stopCard(req *StopReqJSON) (*commons.RespSingleInt, error) {

	// set default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_DO_NOT_HONOR)

	// check function parameters
	if req == nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + " invalid parameter req=nil")
	}
	if req.Reference == "" {
		return resp, fmt.Errorf(utils.GetFunctionName() + " invalid reference or amount parameter")
	}

	// check that the card is valid for operations
	card, err:= commons.GetCard(req.Reference, req.VoucherNumber[len(req.VoucherNumber)-4:len(req.VoucherNumber)])
	if err != nil || card == nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		// return default response
		return resp, nil
	}

	// check that the card is not already stopped
	if card.StatusId == commons.CARD_STATUS_STOPPED {
		return commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED), nil
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// update card status
	ct, err := tx.Exec(ctx, "UPDATE card_issued SET status_id = $1 WHERE card_id = $2",
			commons.CARD_STATUS_STOPPED, card.CardId)
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
	ct, err = tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, req.Reference, STOP_TX_TYPE, commons.TX_OPER_INFO, 0,	
		commons.RESP_CODE[commons.RESP_CODE_APPROVED] + " - " + " CARD " + req.VoucherNumber + " HAS BEEN STOPPED",
		string(jsonStr))
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

	// set response to approved
	resp = commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED)
	return resp, nil
}
