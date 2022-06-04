// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package db provides database connection services
package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/gastonstec/autho/gojlogger"
	"github.com/gastonstec/autho/config"
)

// Connection pool
var DBRead *pgxpool.Pool
var DBWrite *pgxpool.Pool


// Opens database connections pools
func OpenDB() error {
	var err error

	// create read connection
	DBRead, err = pgxpool.Connect(context.Background(), config.ConnStrRead)
	if err != nil {
		return err
	}
	err = checkConnection(DBRead)
	if err != nil {
		return err
	}

	// create read-write connection
	DBWrite, err = pgxpool.Connect(context.Background(), config.ConnStrWrite)
	if err != nil {
		return err
	}
	err = checkConnection(DBWrite)
	if err != nil {
		return err
	}

	return nil
}

// Function checkConnection test a connection and log the result
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
	gojlogger.LogInfo(fmt.Sprintf("Connected to dbname=%s version=%s on server=%s port=%s with user=%s",
					dbname, version, server, port, dbconn.Config().ConnConfig.User))

	return nil
}


// Closes the connection pool
func CloseDB() {
	DBRead.Close()
	DBWrite.Close()
	gojlogger.LogInfo("Database connections has been closed")
}
