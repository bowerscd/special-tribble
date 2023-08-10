package mealbot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type mealbotSQL struct {
	Database
	sqlConn *sql.DB
}

const usersTable = `
	Users(
		ID INTEGER PRIMARY KEY AUTOINCREMENT,
		Username TEXT UNIQUE NOT NULL
	)
`

const recieptsTable = `
	Reciepts(
		ID INTEGER PRIMARY KEY AUTOINCREMENT,
		Date INTEGER,
		Credits INTEGER,
		Payer TEXT NOT NULL,
		Recipient TEXT NOT NULL,
		FOREIGN KEY (Payer) REFERENCES Users (Username)
		FOREIGN KEY (Recipient) REFERENCES Users (Username)
	)
`

// Initialize the new sqlite database
func (db *mealbotSQL) Init(dbx string) error {
	sqlConn, err := sql.Open("sqlite3", dbx)
	if err != nil {
		return err
	}

	_, err = sqlConn.Exec("PRAGMA foreign_keys=on;")
	if err != nil {
		return err
	}

	// Not quite a prepared statement, but ok
	// because we control all the input
	stmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v;", usersTable)
	_, err = sqlConn.Exec(stmt)
	if err != nil {
		return err
	}

	stmt = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v;", recieptsTable)
	_, err = sqlConn.Exec(stmt)
	if err != nil {
		return err
	}

	db.sqlConn = sqlConn

	return nil
}

func (db *mealbotSQL) Close() {
	db.sqlConn.Close()
}

func (_db *mealbotSQL) GetLegacyDatabase() ([]byte, error) {
	ldb := &db{
		Users:    make([]user, 0),
		Reciepts: make([]reciept, 0),
	}

	users, err := _db.GetUsers()
	if err != nil {
		return nil, err
	}

	reciepts, err := _db.GetAllRecords()
	if err != nil {
		return nil, err
	}

	lookup := make(map[string]int)

	for i, u := range users {
		ldb.Users = append(ldb.Users, user{
			ID:  uint(i),
			UPN: u.Username,
		})

		lookup[u.Username] = i
	}

	for _, r := range reciepts {
		ldb.Reciepts = append(ldb.Reciepts, reciept{
			Payer:    uint(lookup[r.Payer]),
			Payee:    uint(lookup[r.Recipient]),
			NumMeals: r.Credits,
			DateTime: r.Date,
		})
	}

	return json.Marshal(ldb)
}

func (db *mealbotSQL) GetUser(username string) (*Account, error) {
	stmt, err := db.sqlConn.Prepare(`
		SELECT
			Username
		FROM
			Users
		WHERE
			Username = ?;
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	u := &Account{}
	err = stmt.QueryRow(username).Scan(&u.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoUser
	}

	return u, nil
}

func (db *mealbotSQL) GetUserByID(id uint) (*Account, error) {
	stmnt, err := db.sqlConn.Prepare(`
		SELECT
			Username
		From
			Users
		Where
			ID = ?
	`)
	if err != nil {
		return nil, err
	}

	acct := &Account{}

	err = stmnt.QueryRow(id).Scan(&acct.Username)
	if err != nil {
		return nil, err
	}

	return acct, nil
}

func (db *mealbotSQL) GetUsers() ([]Account, error) {
	stmt, err := db.sqlConn.Prepare(`
		SELECT
			Username
		FROM
			Users
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	u := make([]Account, 0)
	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		act := Account{}
		err = rows.Scan(&act.Username)
		if err != nil {
			return nil, err
		}

		u = append(u, act)
	}

	return u, nil
}

func (db *mealbotSQL) CreateUser(username string) error {
	_, err := db.GetUser(username)
	if !errors.Is(err, ErrNoUser) {
		return ErrUserExists
	}

	tx, err := db.sqlConn.Begin()
	if err != nil {
		return err
	}

	stmnt, err := tx.Prepare(`
		INSERT INTO
			Users(Username)
		VALUES
			(?);
		`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmnt.Close()

	_, err = stmnt.Exec(username)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (db *mealbotSQL) CreateRecord(payer, recipient string, credits uint) error {
	// Optimize for good callers
	tx, err := db.sqlConn.Begin()
	if err != nil {
		return err
	}

	stmnt, err := tx.Prepare(`
		INSERT INTO
			Reciepts(Payer, Recipient, Credits, Date)
		VALUES
			(?, ?, ?, ?);`)
	if err != nil {
		return err
	}

	_, err = stmnt.Exec(payer, recipient, credits, time.Now().UTC().Unix())
	if err == nil {
		return tx.Commit()
	}
	tx.Rollback()

	_, err = db.GetUser(payer)
	if err != nil {
		if errors.Is(err, ErrNoUser) {
			return ErrPayerDoesNotExist
		}

		return err
	}

	_, err = db.GetUser(recipient)
	if err != nil {
		if errors.Is(err, ErrNoUser) {
			return ErrRecipientDoesNotExist
		}
		return err
	}

	return nil
}

func (mb *mealbotSQL) GetRecords(limit uint) ([]Record, error) {
	return getRecords(mb, limit)
}

func (mb *mealbotSQL) GetAllRecords() ([]Record, error) {
	return getAllRecords(mb)
}

func (db *mealbotSQL) GetTimeboundRecords(limit uint, start, end time.Time) ([]Record, error) {
	stmnt, err := db.sqlConn.Prepare(`
		SELECT
			Payer, Recipient, Credits, Date
		FROM
			Reciepts
		WHERE
			Date BETWEEN ? AND ?
		ORDER BY Date DESC
		LIMIT ?;
		`)
	if err != nil {
		return nil, err
	}

	rows, err := stmnt.Query(start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Record, 0)
	for rows.Next() {
		r := Record{}
		var unix int64

		err = rows.Scan(&r.Payer, &r.Recipient, &r.Credits, &unix)
		if err != nil {
			return nil, err
		}
		r.Date = time.Unix(unix, 0).UTC()

		result = append(result, r)
	}

	return result, nil
}

func (db *mealbotSQL) GetAllRecordsForUser(user string) ([]Record, error) {
	return getAllRecordsForUser(db, user)
}

func (db *mealbotSQL) GetRecordsForUser(user string, limit uint) ([]Record, error) {
	return getRecordsForUser(db, user, limit)
}

func (db *mealbotSQL) GetTimeboundRecordsForUser(user string, limit uint, start, end time.Time) ([]Record, error) {
	stmnt, err := db.sqlConn.Prepare(`
		SELECT
			Payer, Recipient, Credits, Date
		FROM
			Reciepts
		WHERE
			(
				Payer = ?
					OR
				Recipient = ?
			)
				AND
			Date BETWEEN ? AND ?
		ORDER BY Date DESC
		LIMIT ?;
		`)
	if err != nil {
		return nil, err
	}

	rows, err := stmnt.Query(user, user, start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Record, 0)
	for rows.Next() {
		r := Record{}
		var unix int64

		err = rows.Scan(&r.Payer, &r.Recipient, &r.Credits, &unix)
		if err != nil {
			return nil, err
		}
		r.Date = time.Unix(unix, 0).UTC()

		result = append(result, r)
	}

	return result, nil
}

func (db *mealbotSQL) GetRecordsBetweenUsers(user1, user2 string, limit uint) ([]Record, error) {
	return getRecordsBetweenUsers(db, user1, user2, limit)
}

func (db *mealbotSQL) GetAllRecordsBetweenUsers(user1, user2 string) ([]Record, error) {
	return getAllRecordsBetweenUsers(db, user1, user2)
}

func (db *mealbotSQL) GetSummary() (map[string]map[string]*SummaryRecord, error) {
	return getSummary(db)
}

func (db *mealbotSQL) GetTimeboundRecordsBetweenUsers(user1, user2 string, limit uint, start, end time.Time) ([]Record, error) {
	stmnt, err := db.sqlConn.Prepare(`
		SELECT
			Payer, Recipient, Credits, Date
		FROM
			Reciepts
		WHERE
			(
				(
					Payer = ?
						AND
					Recipient = ?
				)
						OR
				(
					Recipient = ?
						AND
					Payer = ?
				)
			)
				AND
			Date BETWEEN ? AND ?
		ORDER BY Date DESC
		LIMIT ?;
		`)
	if err != nil {
		return nil, err
	}

	rows, err := stmnt.Query(user1, user2, user1, user2, start.Unix(), end.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Record, 0)
	for rows.Next() {
		r := Record{}
		var unix int64

		err = rows.Scan(&r.Payer, &r.Recipient, &r.Credits, &unix)
		if err != nil {
			return nil, err
		}
		r.Date = time.Unix(unix, 0).UTC()

		result = append(result, r)
	}

	return result, nil
}

func (db *mealbotSQL) GetSummaryForUser(user string) (map[string]*SummaryRecord, error) {
	return getSummaryForUser(db, user)
}

func (db *mealbotSQL) GetTimeboundSummaryForUser(user string, start, end time.Time) (map[string]*SummaryRecord, error) {
	users, err := db.GetUsers()
	if err != nil {
		return nil, err
	}

	rv := make(map[string]*SummaryRecord)
	for _, u := range users {
		rv[u.Username] = &SummaryRecord{}
	}

	stmntPaid, err := db.sqlConn.Prepare(`
		SELECT
			Recipient, sum(Credits)
		FROM
			Reciepts
		WHERE
			Payer = ?
				AND
			Date BETWEEN ? AND ?
		GROUP BY Recipient;
		`)
	if err != nil {
		return nil, err
	}

	rowsPaid, err := stmntPaid.Query(user, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rowsPaid.Close()

	for rowsPaid.Next() {
		var recipient string
		var credits uint

		err = rowsPaid.Scan(&recipient, &credits)
		if err != nil {
			return nil, err
		}

		rv[recipient].OutgoingCredits = credits
	}

	stmntRecieved, err := db.sqlConn.Prepare(`
		SELECT
			Payer, sum(Credits)
		FROM
			Reciepts
		WHERE
			Recipient = ?
				AND
			Date BETWEEN ? AND ?
		GROUP BY Payer
		`)
	if err != nil {
		return nil, err
	}

	rowsRecieved, err := stmntRecieved.Query(user, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rowsRecieved.Close()

	for rowsRecieved.Next() {
		var payer string
		var credits uint

		err = rowsRecieved.Scan(&payer, &credits)
		if err != nil {
			return nil, err
		}

		rv[payer].IncomingCredits = credits
	}

	return rv, nil
}
