package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/bowerscd/special-tribble/api"
	"github.com/bowerscd/special-tribble/mealbot"
)

//go:generate $GOPATH/bin/oapi-codegen --package=api_test -generate=client,types -o ./client.gen_test.go v1.yml

const (
	TEST_USER1       = "user1"
	TEST_USER2       = "user2"
	NONEXISTENT_USER = "user3"
)

func createUser(client ClientInterface, username string) (*http.Response, error) {
	op := CREATE
	param := PostApiV1UserJSONRequestBody{
		Operation: &op,
		User:      &username,
	}

	return client.PostApiV1User(context.Background(), param)
}

func TestScenario_Standard(t *testing.T) {
	mb, err := mealbot.Create(mealbot.SQLITE_BACKEND)
	if err != nil {
		t.Fatal(err)
	}

	err = mb.Init(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	ServerInterface := &api.ApiHandler{
		Backend: mb,
	}

	server := httptest.NewServer(api.Handler(ServerInterface))
	defer server.Close()

	client, err := NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Create test users
	resp, err := createUser(client, TEST_USER1)
	if err != nil {
		t.Fatal(err)
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Error("unexpected error code / response")
	}

	// Create test users
	resp, err = createUser(client, TEST_USER2)
	if err != nil {
		t.Fatal(err)
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Error("unexpected error code / response")
	}

	summParam := &GetApiV1SummaryParams{nil, nil, nil}
	r2, err := client.GetApiV1SummaryWithResponse(context.Background(), summParam)
	if err != nil {
		t.Fatal(err)
	}

	if r2 == nil || r2.StatusCode() != http.StatusOK {
		t.Error("unexpected error code / response")
	}

	// Ensure a sumamry includes 0s for all users
	summ := GlobalRecordSummary{}
	err = json.Unmarshal(r2.Body, &summ)
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
	recordResp, err := client.PostApiV1RecordWithResponse(context.Background(), CreateRecordParam{
		Credits:   1,
		Payer:     TEST_USER1,
		Recipient: TEST_USER2,
	})
	if err != nil || recordResp == nil || recordResp.StatusCode() != http.StatusOK {
		t.Fatal(err)
	}

	// Ensure the summary is updated
	summParam = &GetApiV1SummaryParams{nil, nil, nil}
	r2, err = client.GetApiV1SummaryWithResponse(context.Background(), summParam)
	if err != nil {
		t.Fatal(err)
	}

	if r2 == nil || r2.StatusCode() != http.StatusOK {
		t.Error("unexpected error code / response")
	}

	// Ensure a sumamry includes 0s for all users
	summ = GlobalRecordSummary{}
	err = json.Unmarshal(r2.Body, &summ)
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

	// Add a debt
	recordResp, err = client.PostApiV1RecordWithResponse(context.Background(), CreateRecordParam{
		Credits:   1,
		Payer:     TEST_USER2,
		Recipient: TEST_USER1,
	})
	if err != nil || recordResp == nil || recordResp.StatusCode() != http.StatusOK {
		t.Fatal(err)
	}

	// r, err := mb.GetAllRecordsForUser(TEST_USER1)
	u1 := TEST_USER1
	p := GetApiV1RecordParams{User1: &u1}
	r3, err := client.GetApiV1RecordWithResponse(context.Background(), &p)
	if err != nil || r3 == nil || r3.StatusCode() != http.StatusOK {
		t.Fatal(err)
	}

	rr1 := []Record{}
	err = json.Unmarshal(r3.Body, &rr1)
	if err != nil {
		t.Fatal(err)
	}

	if len(rr1) != 2 {
		t.Fatalf("record dimensions did not match expected, got %v", len(rr1))
	}

	if rr1[0].Payer != TEST_USER1 || rr1[0].Recipient != TEST_USER2 || rr1[0].Credits != 1 {
		t.Fatalf("first record did not match expected %v", rr1[0])
	}

	if rr1[1].Payer != TEST_USER2 || rr1[1].Recipient != TEST_USER1 || rr1[1].Credits != 1 {
		t.Fatalf("second record did not match expected %v", rr1[1])
	}

	p = GetApiV1RecordParams{}
	r4, err := client.GetApiV1RecordWithResponse(context.Background(), &p)
	if err != nil || r4 == nil || r4.StatusCode() != http.StatusOK {
		t.Fatal(err)
	}

	rr2 := []Record{}
	err = json.Unmarshal(r4.Body, &rr2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(rr1, rr2) {
		t.Fatalf("%v != %v", rr1, rr2)
	}

	u2 := TEST_USER2
	p = GetApiV1RecordParams{
		User1: &u1,
		User2: &u2,
	}

	r5, err := client.GetApiV1RecordWithResponse(context.Background(), &p)
	if err != nil || r5 == nil || r5.StatusCode() != http.StatusOK {
		t.Fatal(err)
	}

	rr3 := []Record{}
	err = json.Unmarshal(r5.Body, &rr3)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(rr2, rr3) || !reflect.DeepEqual(rr1, rr3) {
		t.Fatalf("%v != %v != %v", rr1, rr2, rr3)
	}

	resp, err = client.PostApiV1Record(context.Background(), CreateRecordParam{
		Credits:   1,
		Payer:     TEST_USER1,
		Recipient: NONEXISTENT_USER,
	})

	if resp == nil || err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 got %v", resp.StatusCode)
	}

	resp, err = client.PostApiV1Record(context.Background(), CreateRecordParam{
		Credits:   1,
		Payer:     NONEXISTENT_USER,
		Recipient: TEST_USER1,
	})

	if resp == nil || err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 got %v", resp.StatusCode)
	}

	// err = mb.CreateRecord(TEST_USER1, NONEXISTENT_USER, 1)
	// if err == nil {
	// 	t.Error("expected an error")
	// } else {
	// 	if !errors.Is(err, mealbot.ErrNoUser) {
	// 		t.Errorf("incorrect error: %v", err)
	// 	}
	// }
}
