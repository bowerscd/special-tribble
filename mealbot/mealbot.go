package mealbot

import (
	"errors"
	"math"
	"time"
)

type BACKEND_TYPE int

const (
	JSON_BACKEND   BACKEND_TYPE = 0
	SQLITE_BACKEND BACKEND_TYPE = 1
)

// Function is not implemented
var ErrNotImplemented = errors.New("not implemented")

// User already exists
var ErrUserExists = errors.New("user already exists")

// No such user
var ErrNoUser = errors.New("no such user")
var ErrPayerDoesNotExist = errors.Join(ErrNoUser, errors.New("payer"))
var ErrRecipientDoesNotExist = errors.Join(ErrNoUser, errors.New("recipient"))

// Start() wasn't called before using the DB, or Stop() was called before
// using the DB
var ErrNoActiveDb = errors.New("there is no active database connection")

// User of the mealbot
type Account struct {
	// The unique username for the user
	Username string
}

// Record for a payment between Payer and Recipient.
type Record struct {
	// Who paid
	Payer string

	// Who recieved
	Recipient string

	// How many credits in this transaction.
	Credits int

	// When this transaction occurred.
	Date time.Time
}

// SummaryRecord: represents a summary for a individual.
type SummaryRecord struct {

	// Credits that this entity is owed
	IncomingCredits uint `json:"incoming-credits"`

	// Credits that this entity owes.
	OutgoingCredits uint `json:"outgoing-credits"`
}

type Database interface {
	Init(string) error

	// Close: Gracefully terminate the internal database handler
	// for the mealbot.
	//
	// This will ensure that all file handles are closed, all pending
	// writes are flushed, ensuring the safety & consistency of the
	// database
	Close()

	GetLegacyDatabase() ([]byte, error)

	GetUser(username string) (*Account, error)
	GetUserByID(id uint) (*Account, error)
	GetUsers() ([]Account, error)

	CreateUser(username string) error
	CreateRecord(payer, recipient string, credits uint) error

	// GetRecords: get the last `limit` records
	GetRecords(limit uint) ([]Record, error)

	// GetAllRecords: Get all Records
	GetAllRecords() ([]Record, error)

	// GetTimeboundRecords: Get up to limit Records created during
	// the time interval [Start, End], ordered by date. If all Records
	// are needed, -1 may be passed. This may be a very long API call,
	// if -1 is used.
	//
	// To properly paginate the Records, use the final returned value's
	// time period as the start call to a subsequent call to this API.
	GetTimeboundRecords(limit uint, start, end time.Time) ([]Record, error)

	// Get all records for a specific user
	GetAllRecordsForUser(user string) ([]Record, error)

	// GetRecordsBetweenUsers: Get the last `limit` records that involve `user`
	GetRecordsForUser(user string, limit uint) ([]Record, error)

	// GetRecordsBetweenUsers: Get `limit` records that involve only user
	// between a time specified time period
	GetTimeboundRecordsForUser(user string, limit uint, start, end time.Time) ([]Record, error)

	// GetRecordsBetweenUsers: Get `limit` records that involve only
	// `user1` and `user2`
	GetRecordsBetweenUsers(user1, user2 string, limit uint) ([]Record, error)

	// GetAllRecordsForUser: Get all records for that involve only
	// `user1` and `user2`
	GetAllRecordsBetweenUsers(user1, user2 string) ([]Record, error)

	// GetTimeboundRecordsBetweenUsers: Get up to limit Records created
	// during the time interval [Start, End] that involve both user1
	// and user2.
	GetTimeboundRecordsBetweenUsers(user1, user2 string, limit uint, start, end time.Time) ([]Record, error)

	// GetGlobalSummary: Get a global summary for all users of what debts are
	// owed
	GetSummary() (map[string]map[string]*SummaryRecord, error)

	// GetSummaryForUser: Get a summary for the user of what debts are owed.
	GetSummaryForUser(user string) (map[string]*SummaryRecord, error)

	// GetTimeboundSummaryForUser: Get a summary for the user of what debts
	// are owed by using only Records within the time interval [Start, End].
	GetTimeboundSummaryForUser(user string, start, end time.Time) (map[string]*SummaryRecord, error)
}

func future() time.Time {
	return time.Now().UTC().Add(time.Hour)
}

func past() time.Time {
	return time.Unix(0, 0).UTC()
}

// getSummary: Get a global summary for all users of what debts are
// owed
func getSummary(backend Database) (map[string]map[string]*SummaryRecord, error) {
	users, err := backend.GetUsers()
	if err != nil {
		return nil, err
	}

	results := make(map[string]map[string]*SummaryRecord)
	for _, u := range users {
		sum, err := backend.GetSummaryForUser(u.Username)
		if err != nil {
			return nil, err
		}

		results[u.Username] = sum
	}

	return results, nil
}

func getAllRecords(backend Database) ([]Record, error) {
	return backend.GetTimeboundRecords(math.MaxUint, past(), future())
}

func getRecords(backend Database, limit uint) ([]Record, error) {
	return backend.GetTimeboundRecords(limit, past(), future())
}

func getRecordsForUser(backend Database, user string, limit uint) ([]Record, error) {
	return backend.GetTimeboundRecordsForUser(user, limit, past(), future())
}

func getRecordsBetweenUsers(backend Database, user1, user2 string, limit uint) ([]Record, error) {
	return backend.GetTimeboundRecordsBetweenUsers(user1, user2, math.MaxUint, past(), future())
}

func getAllRecordsBetweenUsers(backend Database, user1, user2 string) ([]Record, error) {
	return backend.GetTimeboundRecordsBetweenUsers(user1, user2, math.MaxUint, past(), future())
}

func getAllRecordsForUser(backend Database, user string) ([]Record, error) {
	return backend.GetTimeboundRecordsForUser(user, math.MaxUint, past(), future())
}

func getSummaryForUser(backend Database, user string) (map[string]*SummaryRecord, error) {
	return backend.GetTimeboundSummaryForUser(user, past(), future())
}

// Create: Start the internal database handler for the mealbot.
func Create(_type BACKEND_TYPE) (Database, error) {
	var backend Database

	switch _type {
	case JSON_BACKEND:
		backend = &mealbotJSON{}
	case SQLITE_BACKEND:
		backend = &mealbotSQL{}
	}

	return backend, nil
}
