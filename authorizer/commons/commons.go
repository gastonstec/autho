// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed
// Package commons contains some global functions and elements
package commons

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/gastonstec/autho/config"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/memdb"
	"github.com/gastonstec/autho/utils"
)

const (
	SOURCE         		= "PMTOL"
	STATUS_SUCCESS 		= "success"
	STATUS_FAIL    		= "fail"
	TX_OPER_WITHDRAW 	= "W"
	TX_OPER_DEPOSIT 	= "D"
	TX_OPER_INFO		= "I"
)

// Paymentology response code constants
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

// card status
const(
		CARD_STATUS_ACTIVE = "ACTIV"
		CARD_STATUS_STOPPED = "STOP"
)

// JSON AuditEvent
type AuditEvent struct {
	TxSource string      `json:"tx-source"`
	TxStatus string      `json:"tx-status"`
	Request  interface{} `json:"request"`
	Response interface{} `json:"response"`
}

// JSON single int response
type RespSingleIntJSON struct {
	ResultCode string `json:"result-code"`
}

// XML single int response
type RespSingleInt struct {
	XMLName   xml.Name               `xml:"methodResponse"`
	TagParams [1]RespSingleIntMember `xml:"params>param>value>struct>member"`
}
type RespSingleIntMember struct {
	Name  string `xml:"name"`
	Value string `xml:"value>int"`
}

// CardInfo struct
type CardInfo struct {
	CardId  		string
	ProviderId 		string
	ProviderCardId	string
	WalletId 		string
	StatusId		string
	UserId 			string
	BIN   			string
	Last4			string
	ExpDate			pgtype.Timestamp
	ValidDate		string
	FirstName		string
	LastName		string
	OtherData		string
}

// Function GetCheckSum builds the checksum value
// Parameters:
//   data = Data to build the checksum value
func GetCheckSum(data string) string {

	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, config.PaymentologyTerminalPasswd)

	// Write Data to it
	h.Write([]byte(data))

	// Get result and encode as hexadecimal string
	cs := hex.EncodeToString(h.Sum(nil))
	cs = strings.ToUpper(cs)

	return cs
}

// Function DecodeKLV transform a klv string to a value map
// Parameters:
//   klv = KLV string to be decoded
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
		if !(mRow == nil) {
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

func MaptoJSON(strmap map[string]string) (string, error) {
	jsonStr, err := json.Marshal(strmap)
	if err != nil {
		return "", err
	}
	return string(jsonStr), nil
}


// Function buildBalanceResp set the struct to map a Balance response
func BuildSingleIntResp(resultCode string) *RespSingleInt {

	// Create struct tag
	methodResp := new(RespSingleInt)

	methodResp.TagParams[0].Name = "resultCode"
	methodResp.TagParams[0].Value = resultCode

	return methodResp
}


// Get a card information
func GetCard(walletID string, last4 string) (*CardInfo, error) {
	var err error

	card := new(CardInfo)

	// get the card
	row := db.DBRead.QueryRow(context.Background(),
		`SELECT card_issued.card_id, usr.user_id, card_issued.status_id, card_issued.expiration_date
		FROM	wallet, "user" usr, wallet_group, card_issued, card_bin
		WHERE	wallet.user_id = usr.user_id
		AND		wallet.group_id = wallet_group.group_id
		AND		wallet.wallet_id = card_issued.wallet_id
		AND		card_issued.bin_number = card_bin.bin_number
		AND 	wallet.wallet_id = $1
		AND		card_issued.last_digits = $2
		AND		card_bin.status_id = 'ACTIV'
		AND		usr.status_id = 'ACTIV'
		AND		wallet_group.status_id = 'ACTIV'
		AND		wallet.status_id = 'ACTIV'`, 
		walletID, last4)
	err = row.Scan(&card.CardId, &card.UserId, &card.StatusId, &card.ExpDate)
	if err != nil {
		return nil, err
	}

	// check query values
	if card.CardId == "" || card.UserId == "" || 
		card.StatusId == "" || card.ExpDate.Time.GoString() == "" {
		return nil, errors.New(utils.GetFunctionName() + " card values cannot be empty")	
	}

	// check for card expiration
	if time.Now().UTC().After(card.ExpDate.Time) {
		return nil, nil	
	}

	return card, nil
}


// Get a card information
func GetCardByID(cardID string) (*CardInfo, error) {
	var err error

	card := new(CardInfo)

	// get the card
	row := db.DBRead.QueryRow(context.Background(),
		`SELECT card_issued.card_id, card_issued.provider_id, card_issued.provider_card_id, card_issued.wallet_id, 
		wallet.user_id, card_issued.status_id, card_issued.bin_number, card_issued.last_digits, card_issued.expiration_date, 
		card_issued.valid_date, card_issued.cardholder_first_name, card_issued.cardholder_last_name, card_issued.other_data 
		FROM card_issued, wallet, card_bin, "user" usr, wallet_group 
		WHERE card_issued.card_id = $1
		AND card_issued.provider_id = 'PMTOL' 
		AND card_issued.wallet_id = wallet.wallet_id 
		AND card_issued.bin_number = card_bin.bin_number 
		AND wallet.user_id = usr.user_id 
		AND wallet.group_id = wallet_group.group_id 
		AND card_bin.status_id = 'ACTIV' 
		AND usr.status_id = 'ACTIV' 
		AND wallet_group.status_id = 'ACTIV' 
		AND wallet.status_id = 'ACTIV'`, cardID)
	err = row.Scan(&card.CardId, &card.ProviderId, &card.ProviderCardId, &card.WalletId, &card.UserId, &card.StatusId,
		&card.BIN, &card.Last4, &card.ExpDate, &card.ValidDate, &card.FirstName, &card.LastName, &card.OtherData)
	if err != nil {
		return nil, err
	}

	// check query values
	if card.CardId == "" || card.UserId == "" || 
		card.StatusId == "" || card.ExpDate.Time.GoString() == "" {
		return nil, fmt.Errorf((utils.GetFunctionName() + " card values cannot be empty"))
	}

	return card, nil
}