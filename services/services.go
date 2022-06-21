// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package handles the application services
package services

import( 
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger"
	memdb "github.com/kueski-dev/paymentology-paymethods/models/memdb"
)

// performs services initial activities
func Start() error {
	// log starting
	logger.LogInfo("Starting paymentology authorizer services")

	// load in-memory database
	logger.LogInfo("Loading in-memory database...")
	err := memdb.Load()
	if err != nil {
		return err
	}

	// success
	logger.LogInfo("Paymentology authorizer services started successfully")
	return nil
}