package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/bowerscd/mealbot/internal"
)

const dbgHeader string = "x-mealbot-bad-request"

const dbgErrEnoughArgs = "1"
const dbgInvalidParameter = "2"
const dbgUnderflowParameter = "3"
const dbgNoDebt = "4"
const dbgOther = "5"

type apiHandler struct {
	http.Handler
}

func ApiHandler() http.Handler {
	return apiHandler{}
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

func get_data(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(rw, r)
		log.Printf("[HTTP][get-data]Failed Method Check '%s' != 'POST'", r.Method)
		return
	}

	d, err := internal.GetDatabase()
	if err != nil {
		log.Printf("[HTTP][get-data]: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Write(d)
}

func edit_meal(rw http.ResponseWriter, r *http.Request) {
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

	internal.EditMeal(payerUpn, payeeUpn, int(payment))
	rw.WriteHeader(http.StatusOK)
}

func whoami(rw http.ResponseWriter, r *http.Request) {
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

	rw.Write([]byte((internal.Whoami(uint(id)))))
}

func (apiHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	api := http.NewServeMux()

	api.HandleFunc("/echo", echo)
	api.HandleFunc("/get-data", get_data)
	api.HandleFunc("/edit_meal/", edit_meal)
	api.HandleFunc("/whoami/", whoami)

	http.StripPrefix("/api", api).ServeHTTP(rw, r)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	server := http.NewServeMux()
	server.Handle("/api/", ApiHandler())
	server.Handle("/", http.FileServer(http.Dir("site/")))

	err := internal.InitDB("./Database.json")
	if err != nil {
		log.Fatalf("%v", err)
	}

	go func() {
		<-ctx.Done()
		internal.KillDB()
		stop()
		os.Exit(0)
	}()

	err = http.ListenAndServe("0.0.0.0:80", server)
	if err != nil {
		log.Fatalf("%v", err)
	}
}
