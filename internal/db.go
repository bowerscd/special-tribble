package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

var _db *db
var dblock *sync.RWMutex
var syncChan chan bool
var dbSignal bool
var signalChan chan bool

/**
 * InitDB: Initialize the database system.
 *
 * Returns an error on failure, if:
 *		1. A database file was found, but could not be deserialized.
 */
func InitDB(dbFile string) error {
	dblock = new(sync.RWMutex)

	dblock.Lock()
	defer dblock.Unlock()

	_, err := os.Stat(dbFile)

	// Doesn't Exist, create a new one
	_db = new(db)
	_db.Reciepts = make([]reciept, 0)
	_db.Users = make([]user, 0)

	if !errors.Is(err, os.ErrNotExist) {
		b, err := ioutil.ReadFile(dbFile)
		if err != nil {
			return err
		}

		err = json.Unmarshal(b, _db)
		if err != nil {
			return err
		}
	}

	syncChan = make(chan bool, 10)
	signalChan = make(chan bool, 1)
	dbSignal = true

	go func() {
		for dbSignal {

			SyncDb(dbFile)
		}

		signalChan <- true
	}()

	return nil
}

/**
 * KillDB - Kill database worker threads safely
 */
func KillDB() {
	// Die.
	dbSignal = false

	// Queue a final sync
	signalChan <- true

	// Wait for return
	<-signalChan
}

/**
 * SyncDB - Write the database's current state to a database file,
 * 			waiting for the flush and write to complete, atomically
 *			move the new database file over the new database file.
 * @dbFile: Database file to replace
 *
 * Acquires dblock SHARED
 *
 * Returns:
 * n/a
 */
func SyncDb(dbFile string) {
	fmt.Println("[DB] Preparing to Sync DB")
	<-syncChan

	dblock.RLock()
	defer dblock.RUnlock()

	/* Empty Channel */
	for x := 0; x < len(syncChan); x++ {
		<-syncChan
	}

	/* Die */
	b, err := json.Marshal(_db)
	if err != nil {
		log.Panicf("[DB] Failed to marshal JSON db, %v", err)
	}

	f, err := ioutil.TempFile(path.Dir(dbFile), "MealBotServe")
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

	fmt.Println("[DB] Sync'd Successfully")
}

/**
 * lookup_user: Lookup a user in the in-memory database
 * @upn: Unique String-Identifier (i.e. an email) for a user
 *
 * Returns error on failure, or pointer-to-user in database.
 */
func lookup_user(upn string) (*user, error) {
	for _, v := range _db.Users {
		if v.UPN == upn {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("user '%s' not found", upn)
}

/**
 * lookup_record: Lookup a reciept record in the in-memory database
 * @Payer - user who owes money to @Payee
 * @Payee - user who is owed money by @Payer
 *
 * Returns error on failure, or pointer-to-reciept in database.
 */
func lookup_record(Payer *user, Payee *user) (*reciept, error) {
	for _, v := range _db.Reciepts {
		if v.Payee == Payee.ID && v.Payer == Payer.ID {
			return &v, nil
		}
	}

	log.Printf("%v %v", Payer, Payee)

	return nil, fmt.Errorf("record '%s' -> '%s' not found", Payer.UPN, Payee.UPN)
}

/**
 * GetDatabase: Get the entire database
 */
func GetDatabase() ([]byte, error) {
	dblock.RLock()
	defer dblock.RUnlock()

	return json.Marshal(&_db)
}

/**
 * Whoami: Lookup a UPN by ID
 * @ID - ID to lookup
 */
func Whoami(ID uint) string {
	dblock.RLock()
	defer dblock.RUnlock()

	return _db.Users[ID].UPN
}

/**
 * create_user: Create a blank user record in the in-memory database
 * @UPN - Human Identifier (alias) for user
 *
 * Returns pointer-to-user in database.
 */
func create_user(upn string) *user {
	newUser := user{
		uint(len(_db.Users)),
		upn,
	}

	_db.Users = append(_db.Users, newUser)

	return &_db.Users[len(_db.Users)-1]
}

/**
 * create_record: Create a blank reciept record in the in-memory database
 * @Payer - user who owes money to @Payee
 * @Payee - user who is owed money by @Payer
 *
 * Returns pointer-to-reciept in database.
 */
func create_record(Payer *user, Payee *user) *reciept {

	r := reciept{Payee.ID, Payer.ID, 0, time.Now()}

	_db.Reciepts = append(_db.Reciepts, r)

	return &_db.Reciepts[len(_db.Reciepts)-1]
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
func EditMeal(Payer string, Payee string, X int) {
	dblock.Lock()
	defer dblock.Unlock()

	payee, err := lookup_user(Payee)
	if err != nil {
		payee = create_user(Payee)
	}

	payer, err := lookup_user(Payer)
	if err != nil {
		payer = create_user(Payer)
	}

	record := create_record(payer, payee)
	record.NumMeals += X
	syncChan <- true
}

/**
 * CheckDebts: Locate all records of all debts
 * @User - user to check debts of
 *
 * Obtains dblock shared.
 *
 * Returns error on failure
 * @rv - summary of debts
 */
func CheckDebts() *DebtTable {
	dblock.RLock()
	defer dblock.RUnlock()

	rv := new(DebtTable)
	rv.Labels = make(map[string]uint, len(_db.Users))

	for _, v := range _db.Users {
		rv.Labels[v.UPN] = v.ID
	}

	rv.Debts = make([][]int, len(_db.Users))
	for i := range rv.Debts {
		rv.Debts[i] = make([]int, len(rv.Debts))
	}

	for _, v := range _db.Reciepts {
		rv.Debts[v.Payer][v.Payee] = v.NumMeals
	}

	return rv
}
