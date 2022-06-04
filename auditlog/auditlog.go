// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed

// Package auditlog provides application
// audit log services
package auditlog

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/gastonstec/autho/db"
	"github.com/gastonstec/autho/utils"
)

const (
	LIMIT_DEFAULT = 25   // default number of events to get
	LIMIT_MAX     = 1000 // Maximum number of events to get
)

// AuditLog Event Struct
type LogEvent struct {
	ApplicationID  string      `json:"application-id"`
	ComponentID    string      `json:"component-id"`
	EventID        string      `json:"event-id"`
	EventTimestamp string      `json:"event-timestamp"`
	EventData      interface{} `json:"event-data"`
	CreatedAt      string      `json:"created-at"`
	UpdatedAt      *string     `json:"updated-at"`
}

// Function dbGetEvents returns audit log events from the database.
func dbGetEvents(eventID string, limit int) ([]LogEvent, error) {
	var err error
	var events []LogEvent
	var event *LogEvent

	// Select one event from the database
	if eventID != "" && eventID != "*" {

		// Select event from the database
		row := db.DBRead.QueryRow(context.Background(),
			`SELECT application_id, component_id, event_id,
			to_char(event_timestamp::timestamp, 'YYYY-MM-DD hh24:mi:ss') AS event_timestamp, 
			event_data,	to_char(created_at::timestamp,'YYYY-MM-DD hh24:mi:ss') AS created_at, 
			to_char(updated_at::timestamp,'YYYY-MM-DD hh24:mi:ss') AS updated_at 
			FROM application_audit_log WHERE event_id=$1`, eventID)

		// Get event and check for errors
		event = new(LogEvent)
		err = row.Scan(&event.ApplicationID, &event.ComponentID, &event.EventID, &event.EventTimestamp,
			&event.EventData, &event.CreatedAt, &event.UpdatedAt)
		if event.EventID == "" {
			return nil, err
		}

		events = append(events, *event)

		return events, nil
	}

	// Select last events from the database
	var rows pgx.Rows
	rows, err = db.DBRead.Query(context.Background(),
		`SELECT application_id, component_id, event_id,
		to_char(event_timestamp::timestamp, 'YYYY-MM-DD hh24:mi:ss') AS event_timestamp, 
		event_data,	to_char(created_at::timestamp,'YYYY-MM-DD hh24:mi:ss') AS created_at, 
		to_char(updated_at::timestamp,'YYYY-MM-DD hh24:mi:ss') AS updated_at 
		FROM application_audit_log ORDER BY event_timestamp DESC LIMIT $1`, limit)
	// Check for errors
	if err != nil {
		return nil, err
	}
	// Defer rows close
	defer rows.Close()
	// Scan rows
	for rows.Next() {
		event = new(LogEvent)
		err := rows.Scan(&event.ApplicationID, &event.ComponentID, &event.EventID, &event.EventTimestamp,
			&event.EventData, &event.CreatedAt, &event.UpdatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}

	return events, nil
}

// Function PostEvent post an audit event to the database.
// Parameters:
//   appId = assigned application id on "application" table
//   compId = assigned component id
//   eventData = json to be stored
func PostEvent(appId string, compId string, eventData string) (string, error) {
	var err error

	// Check appId parameter
	if appId == "" || compId == "" {
		err = errors.New("auditlog post error - ApplicationId and ComponentId cannot be empty")
		return "", err
	}

	// Check eventData JSON format
	if eventData == "" || !utils.IsJSON(eventData) {
		err = errors.New("auditlog post error - invalid json in event data parameter")
		return "", err
	}

	// Insert event in the database
	row := db.DBWrite.QueryRow(context.Background(),
		"INSERT INTO application_audit_log(application_id, component_id, event_data) VALUES ($1, $2, $3) returning (event_id)",
		appId, compId, eventData)

	// Get assigned eventId and check for errors
	var eventID string
	err = row.Scan(&eventID)
	if eventID == "" {
		return "", err
	}

	return eventID, nil
}

// Function GetEvents gets events from the audit log.
// Parameters:
//   eventID = event id to look for, use "*" or empty for the last events
//   limit = max number of events to return
func GetEvents(eventID string, limit int) ([]LogEvent, error) {

	// Check input parameters
	if limit <= 0 || limit > LIMIT_MAX {
		limit = LIMIT_DEFAULT
	}

	// Get events from the database
	events, err := dbGetEvents(eventID, limit)

	// Check for errors
	if err != nil {
		return nil, err
	}

	return events, nil
}
