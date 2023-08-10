package mealbot_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/bowerscd/special-tribble/mealbot"
)

const (
	TEST_USER1       = "user1"
	TEST_USER2       = "user2"
	NONEXISTENT_USER = "user3"
)

func TestScenario_Standard(t *testing.T) {
	testcases := []struct {
		TestName   string
		Backend    mealbot.BACKEND_TYPE
		BackendArg string
	}{
		{
			"in-memory sql",
			mealbot.SQLITE_BACKEND,
			":memory:",
		},
		{
			"sqlite backend",
			mealbot.SQLITE_BACKEND,
			"/tmp/db.sqlite",
		},
		{
			"json backend",
			mealbot.JSON_BACKEND,
			"/tmp/db.json",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.TestName, func(t *testing.T) {
			if strings.HasPrefix(tt.BackendArg, "/") {
				os.Remove(tt.BackendArg)
			}

			mb, err := mealbot.Create(tt.Backend)
			if err != nil {
				t.Fatal(err)
			}

			// Start instance
			err = mb.Init(tt.BackendArg)
			if err != nil {
				t.Fatal(err)
			}
			defer mb.Close()

			// Create test users
			err = mb.CreateUser(TEST_USER1)
			if err != nil {
				t.Fatal(err)
			}

			err = mb.CreateUser(TEST_USER2)
			if err != nil {
				t.Fatal(err)
			}

			// Ensure a sumamry includes 0s for all users
			summ, err := mb.GetSummary()
			if err != nil {
				t.Fatal(err)
			}

			// contains both
			if _, ok := summ[TEST_USER1]; !ok {
				t.Fatalf("summary does not contain required information for '%v'", TEST_USER1)
			}

			if _, ok := summ[TEST_USER2]; !ok {
				t.Fatalf("summary does not contain required information for '%v'", TEST_USER2)
			}

			if summ[TEST_USER1][TEST_USER2].OutgoingCredits != 0 || summ[TEST_USER1][TEST_USER2].IncomingCredits != 0 || summ[TEST_USER2][TEST_USER1].OutgoingCredits != 0 || summ[TEST_USER2][TEST_USER1].IncomingCredits != 0 {
				t.Fatalf("summary is not correct")
			}

			// Add a debt
			err = mb.CreateRecord(TEST_USER1, TEST_USER2, 1)
			if err != nil {
				t.Fatal(err)
			}

			// Ensure the summary is updated
			summ, err = mb.GetSummary()
			if err != nil {
				t.Fatal(err)
			}

			if _, ok := summ[TEST_USER1]; !ok {
				t.Fatalf("summary does not contain required information for '%v'", TEST_USER1)
			}

			if _, ok := summ[TEST_USER2]; !ok {
				t.Fatalf("summary does not contain required information for '%v'", TEST_USER2)
			}

			if summ[TEST_USER1][TEST_USER2].OutgoingCredits != 1 {
				t.Fatalf("outgoing credits != 1. Did increment occur correctly?")
			}

			if summ[TEST_USER1][TEST_USER2].OutgoingCredits != summ[TEST_USER2][TEST_USER1].IncomingCredits {
				t.Fatalf("summary is not a reflection")
			}

			err = mb.CreateRecord(TEST_USER2, TEST_USER1, 1)
			if err != nil {
				t.Fatal(err)
			}

			r, err := mb.GetAllRecordsForUser(TEST_USER1)
			if err != nil {
				t.Fatal(err)
			}

			if len(r) != 2 {
				t.Fatalf("record dimensions did not match expected, got %v", len(r))
			}

			if r[0].Payer != TEST_USER1 || r[0].Recipient != TEST_USER2 || r[0].Credits != 1 {
				t.Fatalf("first record did not match expected %v", r[0])
			}

			if r[1].Payer != TEST_USER2 || r[1].Recipient != TEST_USER1 || r[1].Credits != 1 {
				t.Fatalf("second record did not match expected %v", r[1])
			}

			r2, err := mb.GetAllRecords()
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(r2, r) {
				t.Fatalf("%v != %v", r, r2)
			}

			r3, err := mb.GetAllRecordsBetweenUsers(TEST_USER1, TEST_USER2)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(r2, r3) || !reflect.DeepEqual(r, r3) {
				t.Fatalf("%v != %v != %v", r, r2, r3)
			}

			err = mb.CreateRecord(NONEXISTENT_USER, TEST_USER1, 1)
			if err == nil {
				t.Error("expected an error")
			} else {
				if !errors.Is(err, mealbot.ErrNoUser) {
					t.Errorf("incorrect error: %v", err)
				}
			}

			err = mb.CreateRecord(TEST_USER1, NONEXISTENT_USER, 1)
			if err == nil {
				t.Error("expected an error")
			} else {
				if !errors.Is(err, mealbot.ErrNoUser) {
					t.Errorf("incorrect error: %v", err)
				}
			}
		})
	}
}

func TestScenario_CloseOpen(t *testing.T) {
	testcases := []struct {
		TestName   string
		Backend    mealbot.BACKEND_TYPE
		BackendArg string
	}{
		{
			"sqlite-backend",
			mealbot.SQLITE_BACKEND,
			"/tmp/db.sqlite",
		},
		{
			"json-backend",
			mealbot.JSON_BACKEND,
			"/tmp/db.json",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.TestName, func(t *testing.T) {
			if strings.HasPrefix(tt.BackendArg, "/") {
				os.Remove(tt.BackendArg)
			}

			mb, err := mealbot.Create(tt.Backend)
			if err != nil {
				t.Fatal(err)
			}

			// Start instance
			err = mb.Init(tt.BackendArg)
			if err != nil {
				t.Fatal(err)
			}

			// Create test users
			err = mb.CreateUser(TEST_USER1)
			if err != nil {
				t.Fatal(err)
			}

			err = mb.CreateUser(TEST_USER2)
			if err != nil {
				t.Fatal(err)
			}

			// Add a debt
			err = mb.CreateRecord(TEST_USER1, TEST_USER2, 1)
			if err != nil {
				t.Fatal(err)
			}

			// Ensure a sumamry includes 0s for all users
			summ, err := mb.GetSummary()
			if err != nil {
				t.Fatal(err)
			}

			if _, ok := summ[TEST_USER1]; !ok {
				t.Fatalf("summary does not contain required information for '%v'", TEST_USER1)
			}

			if _, ok := summ[TEST_USER2]; !ok {
				t.Fatalf("summary does not contain required information for '%v'", TEST_USER2)
			}

			if summ[TEST_USER1][TEST_USER2].OutgoingCredits != 1 {
				t.Fatalf("outgoing credits != 1. Did increment occur correctly?")
			}

			if summ[TEST_USER1][TEST_USER2].OutgoingCredits != summ[TEST_USER2][TEST_USER1].IncomingCredits {
				t.Fatalf("summary is not a reflection")
			}

			mb.Close()

			// reopen
			err = mb.Init(tt.BackendArg)
			if err != nil {
				t.Fatal(err)
			}
			defer mb.Close()

			summ2, err := mb.GetSummary()
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(summ2, summ) {
				t.Fatalf("%v != %v", summ2, summ)
			}
		})
	}
}
