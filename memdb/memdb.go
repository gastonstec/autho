// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed
// Package memdb handles in-memory database
// services
package memdb

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-memdb"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/gojlogger"
)

var imDB *memdb.MemDB

const (
	COMPONENT_NAME = "paymentology authorizer- "
)

// pmtol_klvmap table struct
type KLV struct {
	KeyIndex  string
	KeyName   string
	KeyDescrp string
}

// wallet_tx_type table struct
type WalletTxType struct {
	Id  			string
	GroupStatus  	string
	TxTypeStatus 	string
	TxTypeDescrp   	string
}

// Function createSchema builds the in-memory
// database schema
func CreateSchema() *memdb.DBSchema {
	// Create the DB schema
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			// pmtol_klvmap structure
			"pmtol_klvmap": {
				Name: "pmtol_klvmap",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "KeyIndex"},
					},
					"keyname": {
						Name:    "keyname",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "KeyName"},
					},
					"keydescrp": {
						Name:    "keydescrp",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "KeyDescrp"},
					},
				},
			},
		},
	}

	return schema
}

// Function Load loads the schema tables
func Load() error {
	var err error

	// Create schema
	schema := CreateSchema()
	imDB, err = memdb.NewMemDB(schema)
	if err != nil {
		return err
	}

	// total records counter
	var dbTotalRecords, memdbTotalRecords int64 = 0, 0

	// load pmtol_klvmap table
	dbTotalRecords, memdbTotalRecords, err = loadKLVmap()
	// Check for errors
	if err != nil {
		return err
	}

	// Check loaded records
	if dbTotalRecords > memdbTotalRecords {
		return fmt.Errorf(COMPONENT_NAME + "pmtol_klvmap memory table loaded with fewer database records")
	}

	// Table loaded ok
	gojlogger.LogInfo(fmt.Sprintf(COMPONENT_NAME+"pmtol_klvmap memory table loaded with %d records of %d", memdbTotalRecords, dbTotalRecords))

	return nil
}


// Function GetFirstByIndex gets the first record using
// the id parameter as a key
func GetFirstByIndex(table string, id string) (interface{}, error) {
	var err error
	var row interface{}

	// Create read-only transaction
	txn := imDB.Txn(false)
	defer txn.Abort()

	// Lookup by id
	row, err = txn.First(table, "id", id)
	if err != nil {
		return nil, err
	}

	return row, nil
}


// Function GetAll gets all the records from the
// table parameter value
func GetAll(table string) ([]interface{}, error) {
	var resp []interface{}

	// Create read-only transaction
	txn := imDB.Txn(false)
	defer txn.Abort()

	// List all the people
	rows, err := txn.Get(table, "id")
	if err != nil {
		return nil, err
	}

	// Scan the rows
	for obj := rows.Next(); obj != nil; obj = rows.Next() {
		resp = append(resp, obj)
	}

	return resp, nil
}


// Function LoadKLVmap loads the pmtol_klvmap table
func loadKLVmap() (int64, int64, error) {
	var dbRecords, memRecords int64 = 0, 0
	var err error

	// get the total number of records
	row := db.DBRead.QueryRow(context.Background(), "SELECT count(key_index) FROM pmtol_klvmap")
	err = row.Scan(&dbRecords)
	if dbRecords <= 0 {
		return dbRecords, 0, err
	}

	// get records from the database
	rows, err := db.DBRead.Query(context.Background(), "SELECT key_index, key_name, key_descrp FROM pmtol_klvmap ORDER BY key_index")
	if err != nil {
		return dbRecords, memRecords, err
	}
	defer rows.Close()

	// insert records in the memory database
	var klv KLV

	// create a write transaction
	txn := imDB.Txn(true)

	for rows.Next() {

		err = rows.Scan(&klv.KeyIndex, &klv.KeyName, &klv.KeyDescrp)
		if err != nil {
			return dbRecords, memRecords, err
		}

		err = txn.Insert("pmtol_klvmap", klv)
		if err != nil {
			return dbRecords, memRecords, err
		}

		memRecords += 1
	}

	// commit transaction
	txn.Commit()

	return dbRecords, memRecords, nil
}
