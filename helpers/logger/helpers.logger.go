// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package provides a custom logger with
// Info, Warning an Error logging methods.
//
// The default timezone is UTC, use
// TimeZoneUTC = false for system timezone.
//
// Package usage:
//   1. Initialize logger with Start function
//   2. Use LogInfo, LogWarning and LogError with
//      a plain or JSON string
package helpers

import (
	"log"
	"os"
	"time"
	"fmt"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
)


const(
	CANNOT_OPEN_FILE = "%s: cannot open file %s"
	LEVEL_INFO = "INFO"
	LEVEL_WARNING = "WARNING"
	LEVEL_ERROR = "ERROR"
)


var (
	logger     	*log.Logger = nil	// logger pointer
	logSourceName 	string				// service name
)


// Function InitLogger init the logger with the specified path
// and filename (for example ./mylog.log), if the path is empty
// os.Stdout will be used. The log file is open for append.
func Start(filepath string, sourceName string) error {


	logSourceName = sourceName

	// check filepath
	if filepath == "" {
		// log to stdout
		logger = log.New(os.Stdout, "", 0)
	} else {
		// open log file for append
		logFile, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf(CANNOT_OPEN_FILE, helpers.GetFunctionName(), filepath)
		}
		logger = log.New(logFile, "", 0)
	}

	// return no errors
	return nil
}


// Function LogInfo writes a INFO entry in the log
func LogInfo(msg string) {

	// get entry time
	entryTime := time.Now().UTC().String()

	// check for JSON or plain string
	logEntry:= ""
	if helpers.IsJSON(msg) {
		logEntry = fmt.Sprintf(`{"datetime":"%s", "source":"%s", "level":"%s", "message":%s}`,
					entryTime, logSourceName, LEVEL_INFO, msg)
	} else {
		logEntry = fmt.Sprintf(`{"datetime":"%s", "source":"%s", "level":"%s", "message":"%s"}`,
					entryTime, logSourceName, LEVEL_INFO, msg)
	}

	// insert log entry
	logger.Println(logEntry)

}

// Function LogInfo writes a WARNING entry in the log
func LogWarning(msg string) {

	// get entry time
	entryTime := time.Now().UTC().String()

	// check for JSON or plain string
	logEntry:= ""
	if helpers.IsJSON(msg) {
		logEntry = fmt.Sprintf(`{"datetime":"%s", "source":"%s", "level":"%s", "message":%s}`,
					entryTime, logSourceName, LEVEL_WARNING, msg)
	} else {
		logEntry = fmt.Sprintf(`{"datetime":"%s", "source":"%s", "level":"%s", "message":"%s"}`,
					entryTime, logSourceName, LEVEL_WARNING, msg)
	}

	// insert log entry
	logger.Println(logEntry)

}

// Function LogInfo writes an ERROR entry in the log
func LogError(msg string) {

	// get entry time
	entryTime := time.Now().UTC().String()

	// check for JSON or plain string
	logEntry:= ""
	if helpers.IsJSON(msg) {
		logEntry = fmt.Sprintf(`{"datetime":"%s", "source":"%s", "level":"%s", "message":%s}`,
					entryTime, logSourceName, LEVEL_ERROR, msg)
	} else {
		logEntry = fmt.Sprintf(`{"datetime":"%s", "source":"%s", "level":"%s", "message":"%s"}`,
					entryTime, logSourceName, LEVEL_ERROR, msg)
	}

	// insert log entry
	logger.Println(logEntry)
}