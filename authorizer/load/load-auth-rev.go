package load

/* import(
	"fmt"
	"time"
	"errors"
	"kueski.com/payment-methods-authorizer-paymentology/clogger"
	"kueski.com/payment-methods-authorizer-paymentology/authorizer/paymentology/utils"
	"github.com/gofiber/fiber/v2"
)

// Deduct Reversal JSON structs
type LoadAuthRevReqJSON struct {
	MethodName 		string 		`json:"method-name"`
	TerminalId  	string		`json:"terminal-id"`
	Reference  		string		`json:"reference"`
	RequestAmount	string		`json:"request-amount"`
	Narrative  		string		`json:"narrative"`
	TxData  		string		`json:"tx-data"`
	ReferenceID  	string		`json:"reference-id"`
	ReferenceDate  	string		`json:"reference-date"`
	TxID  			string		`json:"tx-id"`
	TxDate  		string		`json:"tx-date"`
}

////////

func loadAuthRevCard(reference string) string {
	// simulated waiting
	time.Sleep(200 * time.Millisecond)
	return "1"
}


// Handles a Deduct Adjustment Request
func loadAuthRevHandler(c *fiber.Ctx) (*RespSingleInt, error) {
	var err error

	// Parse body and check for errors
	req:= new(Req)
	err = c.BodyParser(req)
	if err != nil {
		return nil, err
	}

	// Get values
	reqJS:= new(LoadAuthRevReqJSON)
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


	// Buid checksum data
	checksumData:= reqJS.MethodName + reqJS.TerminalId + reqJS.Reference +
				reqJS.RequestAmount + reqJS.Narrative + reqJS.TxData +
				reqJS.ReferenceID + reqJS.ReferenceDate + reqJS.TxID + reqJS.TxDate

	// Validate checksum
	checksum := req.TagParams.TagParam[9].TagValue.Value
	if utils.GetCheckSum(checksumData) != checksum {
		return nil, errors.New(ERR_INVALID_CHECKSUM)
	}

	// Check transaction and funds
	resCode:= loadAuthRevCard(reqJS.Reference)

	// Build response
	resp:= buildSingleIntResp(resCode)

	// Set result code
	respJS := new(RespSingleIntJSON)
	respJS.ResultCode = resp.TagParams[0].Value

	// Log event
	_, err = logEvent(false, reqJS, respJS)

	if err != nil {
		gojlogger.LogError(fmt.Sprintf("paymentology- load authorization reversal transactionID=%s not registered in the audit log", reqJS.TxID))
	}

	return resp, nil
} */
