package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bowerscd/special-tribble/api"
	"github.com/bowerscd/special-tribble/mealbot"
)

const ID_JSUTHER = 0
const ID_DEDE = 1

func TestLegacyDatabase(t *testing.T) {
	mb, err := mealbot.Create(mealbot.JSON_BACKEND)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	name := f.Name()
	defer f.Close()
	defer os.Remove(name)

	_, err = f.WriteString(sampleDB)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	err = mb.Init(name)
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

	resp, err := client.GetApiGetDataWithResponse(context.Background())
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Error(err)
	}

	if string(resp.Body) != strings.TrimSpace(sampleDB) {
		t.Errorf("not equal %v", string(resp.Body))
	}
}

func TestWhoAmi(t *testing.T) {

	mb, err := mealbot.Create(mealbot.JSON_BACKEND)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	name := f.Name()
	defer f.Close()
	defer os.Remove(name)

	_, err = f.WriteString(sampleDB)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	err = mb.Init(name)
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

	resp, err := client.GetApiWhoamiUserIDWithResponse(context.Background(), ID_JSUTHER)
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Error(err)
	}

	if string(resp.Body) != "jsuther" {
		t.Error(err)
	}
}

func TestEditMeal(t *testing.T) {

	mb, err := mealbot.Create(mealbot.JSON_BACKEND)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	name := f.Name()
	defer f.Close()
	defer os.Remove(name)

	_, err = f.WriteString(sampleDB)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	err = mb.Init(name)
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

	resp, err := client.GetApiGetDataWithResponse(context.Background())
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Error(err)
	}

	resp2, err := client.PostApiEditMealPayerRecipientPaymentWithResponse(context.Background(), "jsuther", "dede", 1)
	if err != nil || resp2.StatusCode() != http.StatusOK {
		t.Error(err)
	}

	resp3, err := client.GetApiGetDataWithResponse(context.Background())
	if err != nil || resp3.StatusCode() != http.StatusOK {
		t.Error(err)
	}

	if string(resp.Body) == string(resp3.Body) {
		t.Error("did nothing")
	}

	lb := LegacyDatabase{}
	err = json.Unmarshal(resp3.Body, &lb)
	if err != nil {
		t.Fatal(err)
	}

	slices.Reverse(*lb.Reciepts)
	last := (*lb.Reciepts)[0]

	if *last.NumMeals != 1 ||
		*last.Payer != ID_DEDE ||
		*last.Payee != ID_JSUTHER ||
		(*last.DateTime).Before(time.Now().Add(time.Second*-20)) {
		t.Errorf("unexpected reciept %v %v %v", *last.Payee, *last.Payer, *last.NumMeals)
	}

	resp4, err := client.PostApiEditMealPayerRecipientPaymentWithResponse(context.Background(), "jsuther", "dede", -1)
	if err != nil || resp4.StatusCode() != http.StatusOK {
		t.Error(err)
	}

	resp5, err := client.GetApiGetDataWithResponse(context.Background())
	if err != nil || resp5.StatusCode() != http.StatusOK {
		t.Error(err)
	}

	if string(resp3.Body) == string(resp5.Body) {
		t.Error("did nothing")
	}

	lb = LegacyDatabase{}
	err = json.Unmarshal(resp5.Body, &lb)
	if err != nil {
		t.Fatal(err)
	}

	slices.Reverse(*lb.Reciepts)
	last = (*lb.Reciepts)[0]

	if *last.NumMeals != 1 ||
		*last.Payer != ID_JSUTHER ||
		*last.Payee != ID_DEDE ||
		(*last.DateTime).Before(time.Now().Add(time.Second*-20)) {
		t.Errorf("unexpected reciept %v %v %v", *last.Payee, *last.Payer, *last.NumMeals)
	}
}
