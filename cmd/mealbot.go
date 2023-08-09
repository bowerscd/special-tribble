package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bowerscd/special-tribble/api"
	"github.com/bowerscd/special-tribble/mealbot"
	"github.com/bowerscd/special-tribble/site"
)

func webServer(mealbot mealbot.Database, addr string) *http.Server {
	s := http.NewServeMux()

	// Static Files
	site.WebRootHandler(s)

	ServerInterface := &api.ApiHandler{
		Backend: mealbot,
	}

	s.Handle("/api/", api.Handler(ServerInterface))

	// the server itself
	server := &http.Server{
		Addr:    addr,
		Handler: s,
	}

	return server
}

func main() {
	db, err := mealbot.Create(mealbot.JSON_BACKEND)
	if err != nil {
		log.Fatal(err)
	}

	dbFile := os.Getenv("MEALBOT_DB")
	if len(dbFile) == 0 {
		dbFile = "./Database.json"
	}

	err = db.Init(dbFile)
	if err != nil {
		panic(err)
	}

	s := webServer(db, "127.0.0.1:8080")

	log.Println("Server Started")
	s.ListenAndServe()
}
