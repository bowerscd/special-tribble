package api

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bowerscd/special-tribble/mealbot"
)

const dbgHeader string = "x-mealbot-bad-request"

const dbgErrEnoughArgs = "1"
const dbgInvalidParameter = "2"

type v0ApiHandler struct {
	http.Handler

	mealbot mealbot.Database
}

func echo(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(rw, r)
		log.Printf("[HTTP][echo]Failed Method Check '%s' != 'POST'", r.Method)
		return
	}

	body := make([]byte, r.ContentLength)
	n, err := r.Body.Read(body)
	if !errors.Is(err, io.EOF) {
		log.Printf("Failed '%v'", err)
	}

	if n != len(body) {
		log.Printf("[HTTP][echo]Size mismatch: %d (content length) vs %d (actual)", len(body), n)
	}

	rw.Write(body)
}

func get_data(mealbot mealbot.Database) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {

		if r.Method != "GET" {
			http.NotFound(rw, r)
			log.Printf("[HTTP][get-data]Failed Method Check '%s' != 'POST'", r.Method)
			return
		}

		d, err := mealbot.GetLegacyDatabase()
		if err != nil {
			log.Printf("[HTTP][get-data]: %v", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Write(d)
	}
}

func edit_meal(mealbot mealbot.Database) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {

		if r.Method != "POST" {
			http.NotFound(rw, r)
			log.Printf("[HTTP][edit-meal]Failed Method Check '%s' != 'POST'", r.Method)
			return
		}

		params := strings.Split(strings.Replace(r.URL.Path, "/edit_meal/", "", -1), "/")

		if len(params) < 3 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Header().Add(dbgHeader, dbgErrEnoughArgs)
			log.Printf("[HTTP][edit-meal]Failed Param Check '%d' < 3", len(params))
			return
		}

		payerUpn := params[0]
		payeeUpn := params[1]
		payment, err := strconv.Atoi(params[2])
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Header().Add(dbgHeader, dbgInvalidParameter)
			log.Printf("[HTTP][edit-meal]Failed to convert payment to uint %v", err)
			return
		}

		if payment >= 0 {
			err = mealbot.CreateRecord(payerUpn, payeeUpn, uint(payment))
		} else {
			err = mealbot.CreateRecord(payeeUpn, payerUpn, uint(payment))
		}
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Header().Add(dbgHeader, dbgInvalidParameter)
			log.Printf("[HTTP][edit-meal]user does not exist %v", err)
			return
		}
		rw.WriteHeader(http.StatusOK)
	}
}

func whoami(mealbot mealbot.Database) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {

		if r.Method != "GET" {
			http.NotFound(rw, r)
			log.Printf("[HTTP][edit-meal]Failed Method Check '%s' != 'POST'", r.Method)
			return
		}

		params := strings.Split(strings.Replace(r.URL.Path, "/whoami/", "", -1), "/")

		if len(params) < 1 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Header().Add(dbgHeader, dbgErrEnoughArgs)
			log.Printf("[HTTP][whoami]Failed Param Check '%d' < 1", len(params))
			return
		}

		id, err := strconv.Atoi(params[0])
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Header().Add(dbgHeader, dbgInvalidParameter)
			log.Printf("[HTTP][whoami]Failed to convert payment to uint %v", err)
			return
		}

		uid, err := mealbot.GetUserByID(uint(id))
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Header().Add(dbgHeader, dbgInvalidParameter)
			log.Printf("[HTTP][whoami]Failed to lookup uint %v", err)
			return
		}
		rw.Write([]byte((uid.Username)))
	}
}

func (l v0ApiHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	api := http.NewServeMux()

	api.HandleFunc("/echo", echo)
	api.HandleFunc("/get-data", get_data(l.mealbot))
	api.HandleFunc("/edit_meal/", edit_meal(l.mealbot))
	api.HandleFunc("/whoami/", whoami(l.mealbot))

	http.StripPrefix("/api", api).ServeHTTP(rw, r)
}

func AddV0Api(mealbot mealbot.Database, server *http.ServeMux) {
	server.Handle("/api/", &v0ApiHandler{
		mealbot: mealbot,
	})
}
