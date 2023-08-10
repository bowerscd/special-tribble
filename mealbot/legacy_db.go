package mealbot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"time"
)

type user struct {
	ID  uint
	UPN string
}

type reciept struct {
	Payer    uint
	Payee    uint
	NumMeals int
	DateTime time.Time
}

type db struct {
	Users    []user
	Reciepts []reciept
}

type mealbotJSON struct {
	db
	Database `json:"-"`

	lock       *sync.RWMutex
	syncChan   chan bool
	signalChan chan bool
}

/***
 * DebtTable - a 2D Table represent all debts known
 * @Labels: mapping of alias -> index in the table
 * @Debts: actual data
 *    @Debts[A][B] = -@Debts[B][A]
 */
type DebtTable struct {
	Labels map[string]uint
	Debts  [][]int
}

/**
 * InitDB: Initialize the database system.
 *
 * Returns an error on failure, if:
 *		1. A database file was found, but could not be deserialized.
 */
func (db *mealbotJSON) Init(dbFile string) error {
	db.lock = &sync.RWMutex{}

	db.lock.Lock()
	defer db.lock.Unlock()

	// Doesn't Exist, create a new one
	db.Reciepts = make([]reciept, 0)
	db.Users = make([]user, 0)

	_, err := os.Stat(dbFile)
	if err == nil {

		b, err := os.ReadFile(dbFile)
		if err != nil {
			return err
		}

		err = json.Unmarshal(b, db)
		if err != nil {
			return err
		}
	}

	db.syncChan = make(chan bool, 10)
	db.signalChan = make(chan bool, 1)

	go func(db *mealbotJSON) {
		for <-db.syncChan {
			db.sync(dbFile)
		}

		db.signalChan <- true
	}(db)

	return nil
}

/**
 * KillDB - Kill database worker threads safely
 */
func (db *mealbotJSON) Close() {

	// Queue a final sync
	db.syncChan <- true
	close(db.syncChan)

	// Wait for return
	<-db.signalChan

	// Close
	close(db.signalChan)

}

/**
 * syncDb - Write the database's current state to a database file,
 * 			waiting for the flush and write to complete, atomically
 *			move the new database file over the new database file.
 * @dbFile: Database file to replace
 *
 * Acquires dblock SHARED
 *
 * Returns:
 * n/a
 */
func (db *mealbotJSON) sync(dbFile string) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	/* Empty Channel */
	for x := 0; x < len(db.syncChan); x++ {
		<-db.syncChan
	}

	/* Die */
	b, err := json.Marshal(db)
	if err != nil {
		log.Panicf("[DB] Failed to marshal JSON db, %v", err)
	}

	f, err := os.CreateTemp(path.Dir(dbFile), "MealBotServe")
	if err != nil {
		log.Printf("[DB] Failed to create temp file, %v", err)
		return
	}

	_, err = f.WriteString(string(b))
	if err != nil {
		f.Close()
		log.Printf("[DB] Failed to write to temp file, %v", err)
		return
	}

	err = f.Sync()
	if err != nil {
		f.Close()
		log.Printf("[DB] Failed to synchronize storage, %v", err)
		return
	}
	f.Close()

	err = os.Rename(f.Name(), dbFile)
	if err != nil {
		log.Printf("[DB] Failed to rename new dbFile, %v", err)
	}

	fmt.Println("[DB] Sync Success")
}

/**
 * lookup_user: Lookup a user in the in-memory database
 * @upn: Unique String-Identifier (i.e. an email) for a user
 *
 * Returns error on failure, or pointer-to-user in database.
 */
func (db *mealbotJSON) lookup_user(upn string) (*user, error) {
	for _, v := range db.Users {
		if v.UPN == upn {
			return &v, nil
		}
	}

	return nil, ErrNoUser
}

/**
 * GetDatabase: Get the entire database
 */
func (db *mealbotJSON) GetDatabase() ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return json.Marshal(&db)
}

/**
 * Whoami: Lookup a UPN by ID
 * @ID - ID to lookup
 */
func (db *mealbotJSON) Whoami(ID uint) string {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.Users[ID].UPN
}

/**
 * create_user: Create a blank user record in the in-memory database
 * @UPN - Human Identifier (alias) for user
 *
 * Returns pointer-to-user in database.
 */
func (db *mealbotJSON) create_user(upn string) *user {
	newUser := user{
		uint(len(db.Users)),
		upn,
	}

	db.Users = append(db.Users, newUser)

	return &db.Users[len(db.Users)-1]
}

/**
 * create_record: Create a blank reciept record in the in-memory database
 * @Payer - user who owes money to @Payee
 * @Payee - user who is owed money by @Payer
 *
 * Returns pointer-to-reciept in database.
 */
func (db *mealbotJSON) create_record(Payer *user, Payee *user) *reciept {

	r := reciept{Payee.ID, Payer.ID, 0, time.Now()}

	db.Reciepts = append(db.Reciepts, r)

	return &db.Reciepts[len(db.Reciepts)-1]
}

/**
 * EditMeal: Locate a record via provided UPNs and increment meal count by X
 * @Payer - UPN of user who owes money to @Payee
 * @Payee - UPN of user who is owed money by @Payer
 * @X - value to decrement
 *
 * Obtains dblock exclusive.
 *
 * Returns error on failure
 */
func (db *mealbotJSON) EditMeal(Payer string, Payee string, X int) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	payee, err := db.lookup_user(Payee)
	if err != nil {
		return err
	}

	payer, err := db.lookup_user(Payer)
	if err != nil {
		return err
	}

	record := db.create_record(payer, payee)
	record.NumMeals += X
	db.syncChan <- true

	return nil
}

/**
 * CheckDebts: Locate all records of all debts
 *
 * Obtains dblock shared.
 *
 * Returns error on failure
 * @rv - summary of debts
 */
func (db *mealbotJSON) CheckDebts() *DebtTable {
	db.lock.RLock()
	defer db.lock.RUnlock()

	rv := new(DebtTable)
	rv.Labels = make(map[string]uint, len(db.Users))

	for _, v := range db.Users {
		rv.Labels[v.UPN] = v.ID
	}

	rv.Debts = make([][]int, len(db.Users))
	for i := range rv.Debts {
		rv.Debts[i] = make([]int, len(rv.Debts))
	}

	for _, v := range db.Reciepts {
		rv.Debts[v.Payer][v.Payee] = v.NumMeals
	}

	return rv
}

func (db *mealbotJSON) GetUser(username string) (*Account, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	for _, u := range db.Users {
		if u.UPN == username {
			return &Account{
				Username: u.UPN,
			}, nil
		}
	}

	return nil, ErrNoUser
}

func (db *mealbotJSON) GetUsers() ([]Account, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	rv := make([]Account, len(db.Users))

	for i, u := range db.Users {
		rv[i].Username = u.UPN
	}

	return rv, nil
}

func (db *mealbotJSON) GetUserByID(id uint) (*Account, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return &Account{
		Username: db.Users[id].UPN,
	}, nil
}

func (db *mealbotJSON) CreateUser(username string) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.create_user(username)
	db.syncChan <- true

	return nil
}

func (db *mealbotJSON) CreateRecord(payer, recipient string, credits uint) error {
	return db.EditMeal(payer, recipient, int(credits))
}

func (db *mealbotJSON) GetTimeboundRecords(limit uint, start, end time.Time) ([]Record, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	rv := make([]Record, 0)

	for _, r := range db.Reciepts {
		if r.DateTime.Before(start) || r.DateTime.After(end) {
			continue
		}

		if uint(len(rv)) == limit {
			return rv, nil
		}

		rv = append([]Record{{
			Payer:     db.Users[r.Payer].UPN,
			Recipient: db.Users[r.Payee].UPN,
			Credits:   r.NumMeals,
			Date:      r.DateTime,
		}}, rv...)
	}

	return rv, nil
}

func (db *mealbotJSON) GetTimeboundRecordsForUser(user string, limit uint, start, end time.Time) ([]Record, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	rv := make([]Record, 0)

	uid, err := db.lookup_user(user)
	if err != nil {
		return nil, err
	}

	for _, r := range db.Reciepts {
		if r.Payee != uid.ID && r.Payer != uid.ID {
			continue
		}

		if r.DateTime.Before(start) || r.DateTime.After(end) {
			continue
		}

		if uint(len(rv)) == limit {
			return rv, nil
		}

		rv = append([]Record{{
			Payer:     db.Users[r.Payer].UPN,
			Recipient: db.Users[r.Payee].UPN,
			Credits:   r.NumMeals,
			Date:      r.DateTime,
		}}, rv...)
	}

	return rv, nil
}

func (db *mealbotJSON) GetTimeboundRecordsBetweenUsers(user1, user2 string, limit uint, start, end time.Time) ([]Record, error) {
	uid1, err := db.lookup_user(user1)
	if err != nil {
		return nil, err
	}

	uid2, err := db.lookup_user(user2)
	if err != nil {
		return nil, err
	}

	db.lock.RLock()
	defer db.lock.RUnlock()

	rv := make([]Record, 0)

	for _, r := range db.Reciepts {
		if !(r.Payee != uid1.ID || r.Payer != uid2.ID) && !(r.Payee != uid2.ID && r.Payer != uid1.ID) {
			continue
		}

		if r.DateTime.Before(start) {
			continue
		}

		if r.DateTime.After(end) {
			continue
		}

		if uint(len(rv)) == limit {
			return rv, nil
		}

		rv = append([]Record{{
			Payer:     db.Users[r.Payer].UPN,
			Recipient: db.Users[r.Payee].UPN,
			Credits:   r.NumMeals,
			Date:      r.DateTime,
		}}, rv...)
	}

	return rv, nil
}

func (db *mealbotJSON) GetTimeboundSummaryForUser(user string, start, end time.Time) (map[string]*SummaryRecord, error) {
	self, err := db.lookup_user(user)
	if err != nil {
		return nil, err
	}

	users, err := db.GetUsers()
	if err != nil {
		return nil, err
	}

	db.lock.RLock()
	defer db.lock.RUnlock()

	rv := make(map[string]*SummaryRecord)
	for _, u := range users {
		rv[u.Username] = &SummaryRecord{}
	}

	for _, r := range db.Reciepts {
		var other string

		if r.DateTime.Before(start) || r.DateTime.After(end) {
			continue
		}

		if r.Payee == self.ID {
			other = db.Users[r.Payer].UPN

			if r.NumMeals >= 0 {
				rv[other].OutgoingCredits += uint(r.NumMeals)
			} else {
				rv[other].IncomingCredits += uint(r.NumMeals)
			}
		}

		if r.Payer == self.ID {
			other = db.Users[r.Payee].UPN

			if r.NumMeals >= 0 {
				rv[other].IncomingCredits += uint(r.NumMeals)
			} else {
				rv[other].OutgoingCredits += uint(r.NumMeals)
			}
		}
	}

	return rv, nil
}

func (db *mealbotJSON) GetSummaryForUser(user string) (map[string]*SummaryRecord, error) {
	return getSummaryForUser(db, user)
}

func (db *mealbotJSON) GetLegacyDatabase() ([]byte, error) {
	return db.GetDatabase()
}

func (db *mealbotJSON) GetRecords(limit uint) ([]Record, error) {
	return getRecords(db, limit)
}

func (db *mealbotJSON) GetAllRecordsForUser(user string) ([]Record, error) {
	return getAllRecordsForUser(db, user)
}

func (db *mealbotJSON) GetAllRecords() ([]Record, error) {
	return getAllRecords(db)
}

func (db *mealbotJSON) GetRecordsForUser(user string, limit uint) ([]Record, error) {
	return getRecordsForUser(db, user, limit)
}

func (db *mealbotJSON) GetRecordsBetweenUsers(user1, user2 string, limit uint) ([]Record, error) {
	return getRecordsBetweenUsers(db, user1, user2, limit)
}

func (db *mealbotJSON) GetAllRecordsBetweenUsers(user1, user2 string) ([]Record, error) {
	return getAllRecordsBetweenUsers(db, user1, user2)
}

func (db *mealbotJSON) GetSummary() (map[string]map[string]*SummaryRecord, error) {
	return getSummary(db)
}
