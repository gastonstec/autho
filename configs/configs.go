// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package provides functionality for handling
// configuration variables
package configs

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/kueski-dev/paymentology-paymethods/helpers"
	logger "github.com/kueski-dev/paymentology-paymethods/helpers/logger"
)

// Paymentology configuration values
var PaymentologyTerminal		string
var PaymentologyTerminalPasswd	[]byte

// AWS configuration values
var AWSRegion = ""
var AWSSecretId = ""

// Database configuration values
var ConnStrRead 				string
var ConnStrWrite 				string
const DB_POOL_MAXCONNS 			int = 50

// Application configuration values
const FiberPort string = ":3000"
const SERVICE_NAME = "paymentology-paymethods"


// Function LoadConfig loads the appplication 
// configuration variables
func LoadConfig() error {

	// get aws secrets variables
	AWSSecretId, ok := os.LookupEnv("SECRET_ID")
	if !ok || AWSSecretId == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", "SECRET_ID environment variable not set")
	}
	AWSRegion, ok := os.LookupEnv("AWS_REGION")
	if !ok || AWSRegion == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", "AWS_REGION environment variable not set")
	}

	awsSecret, err := getAWSSecret(AWSRegion, AWSSecretId)
	if err != nil {
		return fmt.Errorf(helpers.GetFunctionName() + "- getting awsSecret error=%s", err.Error())
	}

	if awsSecret != nil {
		PaymentologyTerminal = awsSecret["paymentology-terminal"]
		PaymentologyTerminalPasswd = []byte(awsSecret["paymentology-terminal-password"])
		logger.LogInfo(fmt.Sprintf(helpers.GetFunctionName() + "- %s", "Paymentology terminal values has been set"))
	} else {
		logger.LogError(fmt.Sprintf(helpers.GetFunctionName() + "- %s", "Paymentology terminal values not set"))
	}

	// db connection variables
	connRead, ok := os.LookupEnv("APP_DB_CONN_READ")
	if !ok || connRead == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", "APP_DB_CONN_READ environment variable not set")
	} else {
		logger.LogInfo(fmt.Sprintf(helpers.GetFunctionName() + "- %s", "APP_DB_CONN_READ environment variable has been set"))
	}
	connWrite, ok := os.LookupEnv("APP_DB_CONN_WRITE")
	if !ok || connWrite == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", "APP_DB_CONN_WRITE environment variable not set")
	} else {
		logger.LogInfo(fmt.Sprintf(helpers.GetFunctionName() + "- %s", "APP_DB_CONN_WRITE environment variable has been set"))
	}

	
	// build connection strings
	ConnStrRead = getConnUrl(connRead)
	if ConnStrRead == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", "connection string for database read cannot be empty")
	}
	ConnStrWrite = getConnUrl(connWrite)
	if ConnStrWrite == "" {
		return fmt.Errorf(helpers.GetFunctionName() + "- %s", "connection string for database read cannot be empty")
	}

	return nil
}


// Function getConnUrl decode the connection url to connection string
func getConnUrl(envVar string) string {

	var dataMap map[string]string
	json.Unmarshal([]byte(envVar), &dataMap)

	dbUrl := fmt.Sprintf("postgres://%v:%v@%v/%v?application_name=%s&pool_max_conns=%d",
			dataMap["user"], url.QueryEscape(dataMap["password"]), dataMap["host_with_port"], dataMap["name"], SERVICE_NAME, DB_POOL_MAXCONNS)
	
	return dbUrl
}


// Get a secret from AWS Secrets Manager
func getAWSSecret(region string, secretId string) (map[string]string, error) {

	const ERROR_FORMAT = "%s - %s - %s"

	// create aws session
	sess, err := session.NewSession()
	if err != nil {
		// Handle session creation error
		return nil, fmt.Errorf("%s - aws session opening error=%s", helpers.GetFunctionName(), err.Error())
	}

	// set secrets manager service
	svc := secretsmanager.New(sess, aws.NewConfig().WithRegion(region))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretId),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// get secret value
	result, err := svc.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
				case secretsmanager.ErrCodeDecryptionFailure:
					// Secrets Manager can't decrypt the protected secret text using the provided KMS key.
					return nil, fmt.Errorf(ERROR_FORMAT, helpers.GetFunctionName(), secretsmanager.ErrCodeDecryptionFailure, aerr.Error())

				case secretsmanager.ErrCodeInternalServiceError:
					// An error occurred on the server side.
					return nil, fmt.Errorf(ERROR_FORMAT, helpers.GetFunctionName(), secretsmanager.ErrCodeInternalServiceError, aerr.Error())

				case secretsmanager.ErrCodeInvalidParameterException:
					// You provided an invalid value for a parameter.
					return nil, fmt.Errorf(ERROR_FORMAT, helpers.GetFunctionName(), secretsmanager.ErrCodeInvalidParameterException, aerr.Error())

				case secretsmanager.ErrCodeInvalidRequestException:
					// You provided a parameter value that is not valid for the current state of the resource.
					return nil, fmt.Errorf(ERROR_FORMAT, helpers.GetFunctionName(), secretsmanager.ErrCodeInvalidRequestException, aerr.Error())

				case secretsmanager.ErrCodeResourceNotFoundException:
					// We can't find the resource that you asked for.
					return nil, fmt.Errorf(ERROR_FORMAT, helpers.GetFunctionName(), secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			return nil, fmt.Errorf("%s - error=%s", helpers.GetFunctionName(), err.Error())
		}
	}

	// Decrypts secret using the associated KMS key.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString string
	if result.SecretString != nil {
		secretString = *result.SecretString
	}

	// DEBUG
	logger.LogInfo(fmt.Sprintf("%s - secretString=%s", helpers.GetFunctionName(), secretString))

	secretMap := make(map[string]string)

	err = json.Unmarshal([]byte(secretString), &secretMap)
	if err != nil {
		return nil, fmt.Errorf("%s - unmarshal aws secret error=%s", helpers.GetFunctionName(), err.Error())
	}

	return secretMap, nil
}