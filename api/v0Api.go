package api

import (
	"io"
	"log"
	"net/http"
)

const dbgHeader string = "x-mealbot-bad-request"

const dbgInvalidParameter = "2"

func (api *ApiHandler) PostApiEcho(rw http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Write(b)
}

func (api *ApiHandler) GetApiGetData(rw http.ResponseWriter, r *http.Request) {
	d, err := api.Backend.GetLegacyDatabase()
	if err != nil {
		log.Printf("[HTTP][get-data]: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Write(d)
}

func (api *ApiHandler) PostApiEditMealPayerRecipientPayment(rw http.ResponseWriter, r *http.Request, payer AccountName, recipient AccountName, payment int32) {
	var err error

	if payment >= 0 {
		err = api.Backend.CreateRecord(payer, recipient, uint(payment))
	} else {
		err = api.Backend.CreateRecord(recipient, payer, uint(-payment))
	}

	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Header().Add(dbgHeader, dbgInvalidParameter)
		log.Printf("[HTTP][edit-meal]user does not exist %v", err)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (api *ApiHandler) GetApiWhoamiUserID(rw http.ResponseWriter, r *http.Request, userID uint32) {
	uid, err := api.Backend.GetUserByID(uint(userID))
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Header().Add(dbgHeader, dbgInvalidParameter)
		log.Printf("[HTTP][whoami]Failed to lookup uint %v", err)
		return
	}

	rw.Write([]byte((uid.Username)))
}
