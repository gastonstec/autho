
// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package have shared functions and data types.
package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"encoding/xml"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	"github.com/kueski-dev/paymentology-paymethods/configs"
	memdb "github.com/kueski-dev/paymentology-paymethods/models/memdb"
)

// Paymentology response code constants
// numeric constants
const (
	RESP_CODE_APPROVED            = "1"
	RESP_CODE_INVALID_CARD        = "-4"
	RESP_CODE_TX_TIMEOUT          = "-7"
	RESP_CODE_AUTHENTICATION_FAIL = "-8"
	RESP_CODE_DO_NOT_HONOR        = "-9"
	RESP_CODE_NOT_SUFF_FUNDS      = "-17"
	RESP_CODE_EXCEEDS_WITHDRAW    = "-18"
	RESP_CODE_INVALID_AMOUNT      = "-19"
	RESP_CODE_SECURITY_VIOLATION  = "-24"
	RESP_CODE_INCORRECT_PIN       = "-25"
	RESP_CODE_PIN_TRIES_EXCEEDED  = "-26"
	RESP_CODE_INVALID_PIN_BLOCK   = "-27"
	RESP_CODE_PIN_LENGTH_ERROR    = "-28"
	RESP_CODE_EXPIRED_CARD        = "-36"
	RESP_CODE_SUSPECTED_FRAUD     = "-37"
	RESP_CODE_LOST_CARD           = "-38"
	RESP_CODE_STOLEN_CARD         = "-39"
)
// description constants
var RESP_CODE = map[string]string{
	"1":   "Approved",
	"-4":  "Invalid card number",
	"-7":  "Transaction timeout",
	"-8":  "Authentication failed",
	"-9":  "Do not honor",
	"-17": "Not sufficent funds",
	"-18": "Exceeds withdrawal amount limit",
	"-19": "Invalid amount",
	"-24": "Security violation",
	"-25": "Incorrect PIN",
	"-26": "Allowable PIN tries exceeded",
	"-27": "Invalid PIN block",
	"-28": "PIN length error",
	"-36": "Expired card",
	"-37": "Suspected fraud",
	"-38": "Lost card",
	"-39": "Stolen card",
}

// Paymentology transaction types
const( 
	TX_TYPE_DEDUCT = "DEDUC"
	TX_TYPE_DEDUCT_ADJUSTMENT = "DEADJ"
	TX_TYPE_DEDUCT_REVERSAL = "DEREV"
	TX_TYPE_LOAD_ADJUSTMENT = "LOADJ"
	TX_TYPE_LOAD_REVERSAL = "LOREV"
	TX_TYPE_LOAD_AUTH_REVERSAL = "LOARE"
	TX_TYPE_LOAD_AUTH = "LOAUT"
	TX_TYPE_CARD_STOP = "CRDST"
)

// Transaction operations
const (
	TX_OPER_INFO		= "I"
)


// Request struct
type Req struct {
	XMLName    xml.Name     `xml:"methodCall"`
	MethodName string       `xml:"methodName"`
	TagParams  ReqParamsTag `xml:"params"`
}
type ReqParamsTag struct {
	TagParam []ReqParam `xml:"param"`
}
type ReqParam struct {
	TagValue ValueStr `xml:"value"`
}
type ValueStr struct {
	Value string `xml:",any"`
}


// XML Request router struct
type XMLReqRouter struct {
	MethodCall xml.Name 	`xml:"methodCall"`
	MethodName string   	`xml:"methodName"`
}


// Request JSON struct
type ReqJSON struct {
	MethodName 		string 				`json:"method-name"`
	TerminalId  	string				`json:"terminal-id"`
	Reference  		string				`json:"reference"`
	RequestAmount	float64				`json:"request-amount"`
	Narrative  		string				`json:"narrative"`
	TxType			string				`json:"tx-type"`
	TxData  		*map[string]string 	`json:"tx-data"`
	TxID  			string				`json:"tx-id"`
	TxDate  		string				`json:"tx-date"`
	Checksum 		string				`json:"checksum"`
}
// Request with reference JSON struct
type ReqWithRefJSON struct {
	MethodName 		string 				`json:"method-name"`
	TerminalId  	string				`json:"terminal-id"`
	Reference  		string				`json:"reference"`
	RequestAmount	float64				`json:"request-amount"`
	Narrative  		string				`json:"narrative"`
	TxData  		*map[string]string 	`json:"tx-data"`
	ReferenceID  	string				`json:"reference-id"`
	ReferenceDate  	string				`json:"reference-date"`
	TxID  			string				`json:"tx-id"`
	TxDate  		string				`json:"tx-date"`
	Checksum 		string				`json:"checksum"`
}
// Stop Card request JSON struct
type StopReqJSON struct {
	MethodName 		string 				`json:"method-name"`
	TerminalId  	string				`json:"terminal-id"`
	Reference  		string				`json:"reference"`
	VoucherNumber 	string				`json:"voucher-number"`
	StopReason 		string				`json:"stop-reason"`
	TxData        	*map[string]string 	`json:"tx-data"`
	TxID  			string				`json:"tx-id"`
	TxDate  		string				`json:"tx-date"`
	Checksum 		string				`json:"checksum"`
}

// Single integer response struct
type RespSingleInt struct {
	XMLName   xml.Name               	`xml:"methodResponse"`
	TagParams [1]RespSingleIntMember 	`xml:"params>param>value>struct>member"`
}
type RespSingleIntMember struct {
	Name  string 						`xml:"name"`
	Value string 						`xml:"value>int"`
}


// Function GetCheckSum builds the checksum value
func GetCheckSum(data string) string {

	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, configs.PaymentologyTerminalPasswd)

	// Write Data to it
	h.Write([]byte(data))

	// Get result and encode as hexadecimal string
	cs := hex.EncodeToString(h.Sum(nil))
	cs = strings.ToUpper(cs)

	return cs
}


// Function DecodeKLV transform a klv string to a value map
func DecodeKLV(klv string) (*map[string]string, error) {
	var err error

	// Create the value map and other variables
	klvmap := make(map[string]string)
	var keylen int
	var keyIndex string
	var mRow interface{}
	var kv memdb.KLV

	// Create map iterating the KLV string
	for i := 0; i < len(klv); {

		// get key index value
		keyIndex = klv[i:(i + 3)]
		// get other values from the memdb and check for error
		mRow, err = memdb.GetFirstByIndex("pmtol_klvmap", keyIndex)
		if err != nil {
			return nil, err
		}

		// check unknown KLV values
		if mRow != nil {
			// map memdb row to KLV struct
			kv = mRow.(memdb.KLV)
		} else {
			// handle unknown KLV
			kv.KeyIndex = keyIndex
			kv.KeyName = "UNKNOWN" + strconv.Itoa(i)
			kv.KeyDescrp = "UNKNOWN"
		}

		// transform memRow to KLV struct
		kv = mRow.(memdb.KLV)

		// transform key length to integer and check for errors
		keylen, err = strconv.Atoi(klv[(i + 3):(i + 5)])
		if err != nil {
			return nil, err
		}
		// Check for no length values and assign value
		if keylen > 0 {
			klvmap[kv.KeyName] = klv[(i + 5):(i + 5 + keylen)]
		} else {
			klvmap[kv.KeyName] = ""
		}

		i = i + 5 + keylen
	}

	return &klvmap, nil
}


// Function StrMaptoJSON transform a string map into a json string
func StrMaptoJSON(strmap map[string]string) (string, error) {
	jsonStr, err := json.Marshal(strmap)
	if err != nil {
		return "", err
	}
	return string(jsonStr), nil
}


// Function BuildSingleIntResp create a single int response
func BuildSingleIntResp(resultCode string) *RespSingleInt {

	// Create struct tag
	methodResp := new(RespSingleInt)

	methodResp.TagParams[0].Name = "resultCode"
	methodResp.TagParams[0].Value = resultCode

	return methodResp
}


// Function RaiseError create a custom error
func RaiseError(source string, msg string) error {
	// build and return error
	return fmt.Errorf(helpers.GetFunctionName() + "- %s", msg)
}


func MapReqToJSON(req *Req) (*ReqJSON, string, error) {
	var err error

	// map values
	reqJS := new(ReqJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	strAmount := req.TagParams.TagParam[2].TagValue.Value
	reqJS.Narrative = req.TagParams.TagParam[3].TagValue.Value
	reqJS.TxType = req.TagParams.TagParam[4].TagValue.Value
	txData := req.TagParams.TagParam[5].TagValue.Value
	reqJS.TxID = req.TagParams.TagParam[6].TagValue.Value
	reqJS.TxDate = req.TagParams.TagParam[7].TagValue.Value
	reqJS.Checksum = req.TagParams.TagParam[8].TagValue.Value

	// convert amount to float
	reqJS.RequestAmount, err = AmountToFloat(strAmount)
	if err != nil {
		return nil, "", RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// decode KLV data
	reqJS.TxData, err = DecodeKLV(txData)
	if err != nil {
		return nil, "", RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// generate checksumdata
	checksumData := reqJS.MethodName + reqJS.TerminalId + reqJS.Reference +
					strAmount + reqJS.Narrative + reqJS.TxType +
					txData + reqJS.TxID + reqJS.TxDate

	
	return reqJS, checksumData, nil
}


func MapReqWithRefToJSON(req *Req) (*ReqWithRefJSON, string, error) {
	var err error

	// map values
	reqJS := new(ReqWithRefJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	strAmount := req.TagParams.TagParam[2].TagValue.Value
	reqJS.Narrative = req.TagParams.TagParam[3].TagValue.Value
	txData := req.TagParams.TagParam[4].TagValue.Value
	reqJS.ReferenceID = req.TagParams.TagParam[5].TagValue.Value
	reqJS.ReferenceDate = req.TagParams.TagParam[6].TagValue.Value
	reqJS.TxID = req.TagParams.TagParam[7].TagValue.Value
	reqJS.TxDate = req.TagParams.TagParam[8].TagValue.Value
	reqJS.Checksum = req.TagParams.TagParam[9].TagValue.Value

	// convert amount to float
	reqJS.RequestAmount, err = AmountToFloat(strAmount)
	if err != nil {
		return nil, "", RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// decode KLV data
	reqJS.TxData, err = DecodeKLV(txData)
	if err != nil {
		return nil, "", RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// generate checksumdata
	checksumData:= reqJS.MethodName + reqJS.TerminalId + reqJS.Reference + 
				strAmount + reqJS.Narrative + txData + reqJS.ReferenceID +
				reqJS.ReferenceDate + reqJS.TxID + reqJS.TxDate
	
	return reqJS, checksumData, nil
}

func MapStopReqToJSON(req *Req) (*StopReqJSON, string, error) {
	var err error

	// map values
	reqJS := new(StopReqJSON)
	reqJS.MethodName = req.MethodName
	reqJS.TerminalId = req.TagParams.TagParam[0].TagValue.Value
	reqJS.Reference = req.TagParams.TagParam[1].TagValue.Value
	reqJS.VoucherNumber = req.TagParams.TagParam[2].TagValue.Value
	reqJS.StopReason = req.TagParams.TagParam[3].TagValue.Value
	txData := req.TagParams.TagParam[4].TagValue.Value
	reqJS.TxID = req.TagParams.TagParam[5].TagValue.Value
	reqJS.TxDate = req.TagParams.TagParam[6].TagValue.Value
	reqJS.Checksum = req.TagParams.TagParam[7].TagValue.Value

	// decode KLV data
	reqJS.TxData, err = DecodeKLV(txData)
	if err != nil {
		return nil, "", RaiseError(helpers.GetFunctionName(), err.Error())
	}

	// generate checksumdata
	checksumData:= reqJS.MethodName + reqJS.TerminalId + reqJS.Reference +
				reqJS.VoucherNumber + reqJS.StopReason + txData +
				reqJS.TxID + reqJS.TxDate
	
	return reqJS, checksumData, nil
}


func MapToJSON(object interface{}) ([]byte, error) {
	// convert request to JSON
	var jsonStr []byte
	jsonStr, err := json.Marshal(object)
	if err != nil {
		return nil, RaiseError(helpers.GetFunctionName(), err.Error())
	}

	return jsonStr, nil
}

func AmountToFloat(amount string) (float64, error){
	// convert amount to float
	fAmount, err := strconv.ParseFloat(amount[0:(len(amount)-2)]+"."+
					amount[(len(amount)-2):], 64)
	if err != nil {
		return 0, fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	return fAmount, nil
}


// Clear (or maybe encrypt) sensitive values comming from requests
func ProtectReqValues(reqJS *ReqJSON) error {
	// clear terminal and checksum values
	reqJS.TerminalId, reqJS.Checksum = "", ""
	return nil
}
func ProtectReqWithRefValues(reqJS *ReqWithRefJSON) error {
	// clear terminal and checksum values
	reqJS.TerminalId, reqJS.Checksum = "", ""
	return nil
}