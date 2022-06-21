// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package handles card entity models
package models

import (
	"context"
	"fmt"
	"time"
	"github.com/jackc/pgtype"
	"github.com/google/uuid"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	"github.com/kueski-dev/paymentology-paymethods/db"
)

// CardInfo struct
type CardInfo struct {
	CardId  			string				`json:"card_id"`
	ProviderId 			string				`json:"provider_id"`
	ProviderCardId		string				`json:"provider_card_id"`
	WalletId 			string				`json:"wallet_id"`
	StatusId			string				`json:"status_id"`
	UserId 				string				`json:"user_id"`
	BIN   				string				`json:"bin_number"`
	Last4				string				`json:"last_digits"`
	ExpDate				pgtype.Timestamp	`json:"expiration_date"`
	ValidDate			string				`json:"valid_date"`
	FirstName			string				`json:"cardholder_first_name"`
	LastName			string				`json:"cardholder_last_name"`
	OtherData			pgtype.JSON			`json:"other_data"`
	BINStatusId			string				`json:"bin_status_id"`
	UserStatusId		string				`json:"user_status_id"`
	WalletGroupStatusId	string				`json:"wallet_group_status_id"`
	WalletStatusId		string				`json:"wallet_status_id"`
}

// bin status
const(
	BIN_STATUS_ACTIVE = "ACTIV"
)

// user status
const(
	USER_STATUS_ACTIVE = "ACTIV"
)

// card status
const(
	CARD_STATUS_ACTIVE = "ACTIV"
	CARD_STATUS_STOPPED = "STOP"
)

// wallet status
const(
	WALLET_GROUP_STATUS_ACTIVE = "ACTIV"
	WALLET_STATUS_ACTIVE = "ACTIV"
)

// Transaction operations
const (
	TX_OPER_INFO		= "I"
)

// Transaction types
const TX_TYPE_CARD_STOP = "CRDST"

// Response codes
const RESP_CODE_APPROVED = "1"
var RESP_CODE = map[string]string{
	"1":   "Approved",
}



// Check if the card is active or expired
func IsActive(card *CardInfo) bool {

	return 	card.BINStatusId == BIN_STATUS_ACTIVE &&
			card.WalletGroupStatusId == WALLET_GROUP_STATUS_ACTIVE &&
			card.WalletStatusId == WALLET_STATUS_ACTIVE &&
			card.UserStatusId == USER_STATUS_ACTIVE &&
			card.StatusId == CARD_STATUS_ACTIVE &&
			card.ExpDate.Time.After(time.Now().UTC())
}


// Get a card information
func GetInfo(walletID string, last4 string) (*CardInfo, error) {
	var err error

	// check parameters
	if 	walletID == "" || last4 == "" {
		return nil, fmt.Errorf(helpers.GetFunctionName() + "- %s", "paramaters cannot be empty")
	}

	card := new(CardInfo)

	// get the card
	row := db.DBRead.QueryRow(context.Background(),
		`SELECT card_issued.card_id, card_issued.provider_id, card_issued.provider_card_id, card_issued.wallet_id, 
		wallet.user_id, card_issued.status_id, card_issued.bin_number, card_issued.last_digits, card_issued.expiration_date, 
		card_issued.valid_date, card_issued.cardholder_first_name, card_issued.cardholder_last_name, card_issued.other_data,
		card_bin.status_id, usr.status_id, wallet_group.status_id, wallet.status_id
		FROM card_issued, wallet, card_bin, "user" usr, wallet_group 
		WHERE	wallet.user_id = usr.user_id
		AND		wallet.group_id = wallet_group.group_id
		AND		wallet.wallet_id = card_issued.wallet_id
		AND		card_issued.provider_id = 'PMTOL'
		AND		card_issued.bin_number = card_bin.bin_number
		AND 	wallet.wallet_id = $1
		AND		card_issued.last_digits = $2`, walletID, last4)
	// get values
	err = row.Scan(&card.CardId, &card.ProviderId, &card.ProviderCardId, &card.WalletId, &card.UserId, &card.StatusId,
		&card.BIN, &card.Last4, &card.ExpDate, &card.ValidDate, &card.FirstName, &card.LastName, &card.OtherData,
		&card.BINStatusId, &card.UserStatusId, &card.WalletGroupStatusId, &card.WalletStatusId)
	if err != nil {
		return nil, fmt.Errorf(helpers.GetFunctionName() + "- card wallet_id=%s last_digits=%s does not exists", walletID, last4)
	}

	// check values
	if card.CardId == "" || card.UserId == "" || 
		card.StatusId == "" || card.ExpDate.Time.GoString() == "" {
		return nil, fmt.Errorf((helpers.GetFunctionName() + " card values cannot be empty"))
	}

	return card, nil
}


// Stop a card
func Stop(walletID string, last4 string, txDescription string, txData string) (error) {
	var err error

	// check parameters
	if 	walletID == "" || last4 == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", "paramaters cannot be empty")
	}

	// get card info
	cardInfo, err := GetInfo(walletID, last4)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// check current card status
	if cardInfo.StatusId == CARD_STATUS_STOPPED {
		return nil
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// update card status to stopped
	ct, err := tx.Exec(ctx, "UPDATE card_issued SET status_id = $1 WHERE card_id = $2",
			CARD_STATUS_STOPPED, cardInfo.CardId)
	if err != nil || ct.String() != "UPDATE 1" {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// insert wallet transaction
	txID := uuid.New().String()
	ct, err = tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, walletID, TX_TYPE_CARD_STOP, TX_OPER_INFO, 0, txDescription, txData)
	if err != nil || ct.String() != "INSERT 0 1" {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// commit database transaction
	err = tx.Commit(ctx)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	return nil
}