// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed
// Package utils provides functionality for reading
// configuration files with json format contents
package config

import (
	"fmt"
	"os"
	"encoding/json"
	"net/url"
)


// Paymentology configuration values
var PaymentologyTerminal		string
var PaymentologyTerminalPasswd	[]byte

// Database configuration values
var ConnStrRead 				string
var ConnStrWrite 				string
const DB_POOL_MAXCONNS 			int = 100

// Application configuration values
const FiberPort string = ":3000"
const ENV = "DEV"


// Function getConnUrlFromEnv decode the connection url to connection string
func getConnUrlFromEnv(envVar string) string {

	var dataMap map[string]string
	json.Unmarshal([]byte(envVar), &dataMap)

	dbUrl := fmt.Sprintf("postgres://%v:%v@%v/%v?application_name=paymentology-paymethods&pool_max_conns=%d", dataMap["user"], url.QueryEscape(dataMap["password"]), 
			dataMap["host_with_port"], dataMap["name"], DB_POOL_MAXCONNS)
	
	return dbUrl
}


// Function LoadConfig loads the configuration variables
func LoadConfig() error {

	// Set development environment values
	if ENV == "DEV" {
		os.Setenv("PAYMENTOLOGY_TERMINAL", "0065482345")
		os.Setenv("PAYMENTOLOGY_TERMINAL_PASSWORD", "58F716BEA8")
		os.Setenv("APP_DB_CONN_READ", `{"user": "postgres","password": "kueski","host": "localhost","host_with_port": "localhost:5433","name": "paymethods","url": "postgres://postgres:c4rec4@localhost:5433/paymethods"}`)
		os.Setenv("APP_DB_CONN_WRITE", `{"user": "postgres","password": "kueski","host": "localhost","host_with_port": "localhost:5433","name": "paymethods","url": "postgres://postgres:c4rec4@localhost:5433/paymethods"}`)
	}

	// Build connection strings
	ConnStrRead = getConnUrlFromEnv(os.Getenv("APP_DB_CONN_READ"))
	ConnStrWrite = getConnUrlFromEnv(os.Getenv("APP_DB_CONN_WRITE"))

	// Get paymentology values
	PaymentologyTerminal = os.Getenv("PAYMENTOLOGY_TERMINAL")
	if (PaymentologyTerminal == ""){
		return fmt.Errorf("PAYMENTOLOGY_TERMINAL value is empty")
	}
	PaymentologyTerminalPasswd = []byte(os.Getenv("PAYMENTOLOGY_TERMINAL_PASSWORD"))
	if (string(PaymentologyTerminalPasswd) == ""){
		return fmt.Errorf("PAYMENTOLOGY_TERMINAL_PASSWORD value is empty")
	}

	return nil
}
