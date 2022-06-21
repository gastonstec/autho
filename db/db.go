// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package provides database connection services
package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger"
	"github.com/kueski-dev/paymentology-paymethods/configs"
)

// Connection pools
var DBRead *pgxpool.Pool		// read pool
var DBWrite *pgxpool.Pool		// write pool


// Opens database connections pools
func OpenDB() error {
	var err error

	// create read connection
	DBRead, err = pgxpool.Connect(context.Background(), configs.ConnStrRead)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- error opening DBRead connection %s", err.Error())
	}
	err = checkConnection(DBRead)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- error testing DBRead connection %s", err.Error())
	}

	// create read-write connection
	DBWrite, err = pgxpool.Connect(context.Background(), configs.ConnStrWrite)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- error opening DBWrite connection %s", err.Error())
	}
	err = checkConnection(DBWrite)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- error testing DBWrite connection %s", err.Error())
	}

	return nil
}

// Test a connection and logs the result
func checkConnection(dbconn *pgxpool.Pool) error {
	var version, dbname, server, port string
	var err error

	// check connection and database info
	row := dbconn.QueryRow(context.Background(), 
			"SELECT version()::TEXT AS version, current_database()::TEXT AS database, inet_server_addr()::TEXT AS server, inet_server_port()::TEXT AS port")
	err = row.Scan(&version, &dbname, &server, &port)
	if err != nil {
		return err
	}
	
	// log database info
	logger.LogInfo(fmt.Sprintf("Connected to dbname=%s version=%s on server=%s port=%s with user=%s",
					dbname, version, server, port, dbconn.Config().ConnConfig.User))

	return nil
}


// Closes the connection pool
func CloseDB() {
	DBRead.Close()
	DBWrite.Close()
	logger.LogInfo("Database connections has been closed")
}
