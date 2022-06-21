// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package handles wallet entity models
package models

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/kueski-dev/paymentology-paymethods/db"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
)

// wallet and user status
const(
	USER_STATUS_ACTIVE = "ACTIV"
	WALLET_STATUS_ACTIVE = "ACTIV"
	WALLET_GROUP_STATUS_ACTIVE = "ACTIV"
)

// general constants
const(
	MSG_APPROVED = "Approved"
	PSQL_MSG_INSERT_1 = "INSERT 0 1"
	PSQL_MSG_UPDATE_1 = "UPDATE 1"
	PSQL_MSG_SET_TX_LEVEL = "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"
	PSQL_MSG_LOCK_WALLET_TABLE = "LOCK TABLE wallet IN ROW EXCLUSIVE MODE"
	PSQL_MSG_LOCK_TABLE = "LOCK TABLE"
)

// Wallet transaction operations
const (
	TX_OPER_WITHDRAW 	= "W"
	TX_OPER_DEPOSIT 	= "D"
	TX_OPER_INFO		= "I"
)

// CardInfo struct
type WalletInfo struct {
	WalletId  				string		`json:"wallet_id"`
	StatusId 				string		`json:"status_id"`
	CurrencyCode 			string		`json:"currency_numeric_code"`
	CurrentBalance 			float64		`json:"current_balance"`
	AvalilableBalance 		float64		`json:"available_balance"`
	BlockedBalance 			float64		`json:"blocked_balance"`
	UserId 					string		`json:"user_id"`
	UserStatusId 			string		`json:"user_status_id"`
	GroupId 				string		`json:"group_id"`
	GroupStatusId			string		`json:"group_status_id"`
}

// Wallet transaction struct
type WalletTransaction struct {
	TransactionId  			string				`json:"transaction_id"`
	WalletId  				string				`json:"wallet_id"`
	GroupId 				string				`json:"group_id"`
	TypeId 					string				`json:"transaction_type_id"`
	Operation 				string				`json:"transaction_operation"`
	Date 					pgtype.Timestamp	`json:"transaction_date"`	
	Amount					float64				`json:"transaction_amount"`
	Description 			string				`json:"transaction_description"`
	Data 					pgtype.JSON			`json:"transaction_data"`
}

const MSG_EMPTY_PARAMETERS = "paramaters cannot be empty"
const MSG_TXDATA_NOT_JSON = "txData is not json"


// Check if the wallet is active or expired
func IsActive(wallet *WalletInfo) bool {

	return 	wallet.UserStatusId == USER_STATUS_ACTIVE &&
			wallet.StatusId == WALLET_STATUS_ACTIVE &&
			wallet.GroupStatusId == WALLET_GROUP_STATUS_ACTIVE
}


// Get a wallet info
func GetInfo(walletID string) (*WalletInfo, error) {

	if 	walletID == "" {
		return nil, fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_EMPTY_PARAMETERS)
	}

	wallet := new(WalletInfo)

	// get the card
	row := db.DBRead.QueryRow(context.Background(),
		`SELECT wallet.wallet_id, wallet.status_id, wallet.currency_numeric_code, wallet.current_balance, wallet.available_balance, 
		wallet.blocked_balance, wallet.user_id, usr.status_id, wallet.group_id, wallet_group.status_id
		FROM 	wallet, "user" usr, wallet_group
		WHERE	wallet.wallet_id = $1 AND wallet.user_id = usr.user_id AND wallet.group_id = wallet_group.group_id`, 
		walletID)

	// get values
	err := row.Scan(&wallet.WalletId, &wallet.StatusId, &wallet.CurrencyCode, &wallet.CurrentBalance, 
				&wallet.AvalilableBalance, &wallet.BlockedBalance, &wallet.UserId, &wallet.UserStatusId,
				&wallet.GroupId, &wallet.GroupStatusId)
	if err != nil {
		return nil, fmt.Errorf(helpers.GetFunctionName() + "- wallet_id=%s does not exists", walletID)
	}

	// check values
	if wallet.WalletId == "" || wallet.UserId == "" {
		return nil, fmt.Errorf((helpers.GetFunctionName() + "- wallet values cannot be empty"))
	}

	return wallet, nil
}


// Get a transaction info
func GetTransaction(walletID string, txId string, externalId bool) (*WalletTransaction, error) {

	walletTX := new(WalletTransaction)

	// build query
	qry:= ""
	if externalId {
		// get using external transaction id
		qry = `SELECT transaction_id, wallet_id, group_id, transaction_type_id, transaction_operation, transaction_date, 
		transaction_amount, transaction_description, transaction_data
		FROM 	wallet_transaction
		WHERE	wallet_transaction.wallet_id = $1 AND wallet_transaction.transaction_data ->> 'tx-id' = $2`
	} else {
		// get using internal transaction id
		qry = `SELECT transaction_id, wallet_id, group_id, transaction_type_id, transaction_operation, transaction_date, 
		transaction_amount, transaction_description, transaction_data
		FROM 	wallet_transaction
		WHERE	wallet_transaction.wallet_id = $1 AND wallet_transaction.transaction_id = $2`
	}

	// get the card
	rows, err := db.DBRead.Query(context.Background(), qry, walletID, txId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// check for results
	if !rows.Next() {
		// no rows
		return nil, nil
	}

	// get values
	rows.Scan(&walletTX.TransactionId, &walletTX.WalletId, &walletTX.GroupId, &walletTX.TypeId, &walletTX.Operation,
				&walletTX.Date, &walletTX.Amount, &walletTX.Description, &walletTX.Data)
	
 	

	// check values
	if walletTX.TransactionId == "" {
		return nil, fmt.Errorf((helpers.GetFunctionName() + "- wallet values cannot be empty"))
	}

	return walletTX, nil
}


// Insert a transaction in the wallet transaction log.
func PostTransaction(walletID string, amount float64, txType string, txOperation string, 
					txDescription string, txData string) (string, error) {

	// check parameters
	if 	walletID == "" || txType == "" || txDescription == "" || txData == "" {
		return "", fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_EMPTY_PARAMETERS)
	}
	if !helpers.IsJSON(txData) {
		return "", fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_TXDATA_NOT_JSON)
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// insert wallet transaction
	txID := uuid.New().String()
	ctag, err := tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, walletID, txType, txOperation, amount, txDescription, txData)
	if err != nil || ctag.String() != PSQL_MSG_INSERT_1 {
		tx.Rollback(ctx)
		return "", fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// commit database transaction
	err = tx.Commit(ctx)
	if err != nil {
		tx.Rollback(ctx)
		return "", fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	return txID, nil
}

// Withdraw amount from available_balance and transfer them to wallet 
// blocked_balance and insert the transaction in the transaction log.
func WithdrawAvailableBalance(walletID string, amount float64, matchBalance float64, 
	txType string, txDescription string, txData string) (error) {

	// check parameters
	if 	walletID == "" || txType == "" || txDescription == "" || txData == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_EMPTY_PARAMETERS)
	}
	if !helpers.IsJSON(txData) {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_TXDATA_NOT_JSON)
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// set transaction isolation level
	cmdt, err := tx.Exec(ctx, PSQL_MSG_SET_TX_LEVEL)
	if err != nil || cmdt.String() != "SET" {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// enable lock wallet table at row level
	cmdt, err = tx.Exec(ctx, PSQL_MSG_LOCK_WALLET_TABLE)
	if err != nil || cmdt.String() != PSQL_MSG_LOCK_TABLE {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// lock wallet row for update
	var availableBal float64
	row := tx.QueryRow(ctx, "SELECT available_balance FROM wallet WHERE wallet_id = $1 FOR UPDATE", walletID)
	err = row.Scan(&availableBal)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// check available balance match
	if availableBal != matchBalance {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", "available balance does not match")
	}

	// update balances on the wallet
	cmdt, err = tx.Exec(ctx, 
		"UPDATE wallet SET available_balance = available_balance - $1, blocked_balance = blocked_balance + $1 WHERE wallet_id = $2 AND available_balance = $3",
		amount, walletID, matchBalance)
	if err != nil || cmdt.String() != PSQL_MSG_UPDATE_1 {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// insert wallet transaction
	txID := uuid.New().String()
	cmdt, err = tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, walletID, txType, TX_OPER_WITHDRAW, amount, txDescription, txData)
	if err != nil || cmdt.String() != PSQL_MSG_INSERT_1 {
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


// Withdraw amount from blocked_balance and insert the transaction
// in the transaction log.
func WithdrawBlockedBalance(walletID string, amount float64, txType string, 
	txDescription string, txData string) (error) {

	// check parameters
	if 	walletID == "" || txType == "" || txDescription == "" || txData == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_EMPTY_PARAMETERS)
	}
	if !helpers.IsJSON(txData) {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_TXDATA_NOT_JSON)
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// set transaction isolation level
	wct, err := tx.Exec(ctx, PSQL_MSG_SET_TX_LEVEL)
	if err != nil || wct.String() != "SET" {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// enable lock wallet table at row level
	wct, err = tx.Exec(ctx, PSQL_MSG_LOCK_WALLET_TABLE)
	if err != nil || wct.String() != PSQL_MSG_LOCK_TABLE {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// lock wallet row for update
	var blockedBal float64
	row := tx.QueryRow(ctx, "SELECT blocked_balance FROM wallet WHERE wallet_id = $1 FOR UPDATE", walletID)
	err = row.Scan(&blockedBal)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// update balances on the wallet
	wct, err = tx.Exec(ctx, 
		"UPDATE wallet SET blocked_balance = blocked_balance - $1 WHERE wallet_id = $2 AND blocked_balance = $3",
		amount, walletID, blockedBal)
	if err != nil || wct.String() != PSQL_MSG_UPDATE_1 {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// insert wallet transaction
	txID := uuid.New().String()
	wct, err = tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, walletID, txType, TX_OPER_INFO, amount, txDescription, txData)
	if err != nil || wct.String() != PSQL_MSG_INSERT_1 {
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


// Deposit amount from blocked_balance and insert the transaction
// in the transaction log.
func DepositBlockedBalance(walletID string, amount float64, txType string, 
	txDescription string, txData string) (error) {

	// check parameters
	if 	walletID == "" || txType == "" || txDescription == "" || txData == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_EMPTY_PARAMETERS)
	}
	if !helpers.IsJSON(txData) {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", MSG_TXDATA_NOT_JSON)
	}

	// begin database transaction
	ctx := context.Background()
	tx, err := db.DBWrite.Begin(ctx)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// set transaction isolation level
	dct, err := tx.Exec(ctx, PSQL_MSG_SET_TX_LEVEL)
	if err != nil || dct.String() != "SET" {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// enable lock wallet table at row level
	dct, err = tx.Exec(ctx, PSQL_MSG_LOCK_WALLET_TABLE)
	if err != nil || dct.String() != PSQL_MSG_LOCK_TABLE {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// lock wallet row for update
	var blockedBal float64
	row := tx.QueryRow(ctx, "SELECT blocked_balance FROM wallet WHERE wallet_id = $1 FOR UPDATE", walletID)
	err = row.Scan(&blockedBal)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// update balances on the wallet
	dct, err = tx.Exec(ctx, 
		"UPDATE wallet SET blocked_balance = blocked_balance + $1 WHERE wallet_id = $2 AND blocked_balance = $3",
		amount, walletID, blockedBal)
	if err != nil || dct.String() != PSQL_MSG_UPDATE_1 {
		tx.Rollback(ctx)
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", err.Error())
	}

	// insert wallet transaction
	txID := uuid.New().String()
	dct, err = tx.Exec(ctx,
		`INSERT INTO wallet_transaction(transaction_id, wallet_id, group_id, transaction_type_id, 
		transaction_operation, transaction_date, transaction_amount, transaction_description, 
		transaction_data, created_at)
		VALUES ($1, $2,'PMTOL', $3, $4, NOW(), $5, $6, $7, NOW())`,
		txID, walletID, txType, TX_OPER_INFO, amount, txDescription, txData)
	if err != nil || dct.String() != PSQL_MSG_INSERT_1 {
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

