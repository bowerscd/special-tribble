package api

import "github.com/bowerscd/special-tribble/mealbot"

type ApiHandler struct {
	Backend mealbot.Database
	ServerInterface
}
