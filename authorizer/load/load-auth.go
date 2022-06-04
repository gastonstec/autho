package load

import(
	"context"
	"encoding/json"
	//"encoding/xml"
	"fmt"
	"strconv"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	// "github.com/jackc/pgconn"
	// "github.com/jackc/pgx/v4"
	"github.com/gastonstec/autho/authorizer/commons"
	"github.com/gastonstec/autho/gojlogger"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/utils"
)

// Deduct Reversal JSON structs
type LoadAuthReqJSON struct {
	MethodName 		string 				`json:"method-name"`
	TerminalId  	string				`json:"terminal-id"`
	Reference  		string				`json:"reference"`
	RequestAmount	string				`json:"request-amount"`
	Narrative  		string				`json:"narrative"`
	TxType			string				`json:"tx-type"`
	TxData  		*map[string]string 	`json:"tx-data"`
	TxID  			string				`json:"tx-id"`
	TxDate  		string				`json:"tx-date"`
}


// Handles a Deduct Adjustment Request
func LoadAuthHandler(c *fiber.Ctx) (*commons.RespSingleInt, error) {
	var err error

	// set a default response
	resp := commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED)

	// Parse body
	req := new(Req)
	err = c.BodyParser(req)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		// return default response
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// Get values
	reqJS:= new(LoadAuthReqJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	reqJS.RequestAmount = req.TagParams.TagParam[2].TagValue.Value
	reqJS.Narrative = req.TagParams.TagParam[3].TagValue.Value
	reqJS.TxType = req.TagParams.TagParam[4].TagValue.Value
	txData := req.TagParams.TagParam[5].TagValue.Value
	reqJS.TxID = req.TagParams.TagParam[6].TagValue.Value
	reqJS.TxDate = req.TagParams.TagParam[7].TagValue.Value

	// Buid checksum data
	checksumData:= reqJS.MethodName + reqJS.TerminalId + reqJS.Reference +
				reqJS.RequestAmount + reqJS.Narrative + reqJS.TxType +
				txData +  reqJS.TxID + reqJS.TxDate

	// Validate checksum
	checksum := req.TagParams.TagParam[8].TagValue.Value
	if commons.GetCheckSum(checksumData) != checksum {
		gojlogger.LogError(utils.GetFunctionName() + " error=invalid checksum value")
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
	
	// process load authorization
	resp, err = loadAuthCard(reqJS)
	if err != nil {
		gojlogger.LogError(utils.GetFunctionName() + " error=" + err.Error())
		// change response to approved
		resp.TagParams[0].Value = commons.RESP_CODE_APPROVED
	}

	// return deduct response
	return resp, nil
}


func loadAuthCard(req *LoadAuthReqJSON) (*commons.RespSingleInt, error) {
	var err error

	resp := commons.BuildSingleIntResp(commons.RESP_CODE_APPROVED)

	// check function parameters
	if req == nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + " invalid parameter req=nil")
	}
	if req.Reference == "" || req.RequestAmount == "" || len(req.RequestAmount) < 2 {
		return resp, fmt.Errorf(utils.GetFunctionName() + " invalid reference or amount parameter")
	}

	// convert amount to float
	var fAmount float64
	fAmount, err = strconv.ParseFloat(req.RequestAmount[0:(len(req.RequestAmount)-2)]+"."+
		req.RequestAmount[(len(req.RequestAmount)-2):(len(req.RequestAmount))], 64)
	if err != nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// convert request to JSON
	var jsonStr []byte
	jsonStr, err = json.Marshal(req)
	if err != nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return resp, fmt.Errorf(utils.GetFunctionName() + ": %s", err.Error())
	}

	
	// insert wallet transaction
	txID := uuid.New().String()
	ct, err := tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', 'LOAUT', $3, NOW(), $4, $5, $6, NOW())`,
		txID, req.Reference, commons.TX_OPER_INFO, fAmount, commons.RESP_CODE[commons.RESP_CODE_APPROVED] + " - " + req.Narrative, string(jsonStr))
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
