package api

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/bowerscd/special-tribble/mealbot"
)

//go:generate $GOPATH/bin/oapi-codegen --package=api -generate=chi-server,types -o ./server.gen.go v1.yml

func (api *ApiHandler) PostApiV1User(rw http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	params := AccountModificationRequest{}
	err = json.Unmarshal(b, &params)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if *params.Operation != CREATE {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = api.Backend.CreateUser(*params.User)
	if err != nil {
		if !errors.Is(err, mealbot.ErrUserExists) {
			rw.WriteHeader(http.StatusBadRequest)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (api *ApiHandler) GetApiV1Summary(rw http.ResponseWriter, r *http.Request, params GetApiV1SummaryParams) {
	var b []byte

	if (params.Start != nil && params.End == nil) || (params.End != nil && params.Start == nil) {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if params.User == nil && params.End == nil && params.Start == nil {
		d, err := api.Backend.GetSummary()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		b, err = json.Marshal(&d)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if params.User != nil && params.End == nil {
		d, err := api.Backend.GetSummaryForUser(*params.User)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		b, err = json.Marshal(&d)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if params.User != nil && params.End != nil {
		start := *params.Start
		end := *params.End

		d, err := api.Backend.GetTimeboundSummaryForUser(*params.User, start, end)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		b, err = json.Marshal(&d)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	rw.Write(b)
}

func (api *ApiHandler) GetApiV1Record(rw http.ResponseWriter, r *http.Request, params GetApiV1RecordParams) {
	var err error
	var end time.Time = time.Now().UTC().Add(time.Hour * 24)
	var start time.Time = time.Unix(0, 0).UTC()
	var res []mealbot.Record
	var limit uint = math.MaxUint32

	// Unqualified time period
	if (params.Start != nil && params.End == nil) || (params.End != nil && params.Start == nil) {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// User2 speciifed without User1
	if params.User2 != nil && params.User1 == nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if params.Limit != nil {
		limit = uint(*params.Limit)
	}

	if params.Start != nil {
		start = *params.Start
		end = *params.End
	}

	if params.User1 == nil && params.User2 == nil {
		// Global Query
		res, err = api.Backend.GetTimeboundRecords(limit, start, end)
	} else if params.User1 != nil {
		res, err = api.Backend.GetTimeboundRecordsForUser(*params.User1, limit, start, end)
	} else {
		res, err = api.Backend.GetTimeboundRecordsBetweenUsers(*params.User1, *params.User2, limit, start, end)
	}

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(res)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Write(b)
}

func (api *ApiHandler) PostApiV1Record(rw http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	params := CreateRecordParam{}
	err = json.Unmarshal(b, &params)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = api.Backend.CreateRecord(params.Payer, params.Recipient, uint(params.Credits))
	if err != nil {
		if errors.Is(err, mealbot.ErrNoUser) || errors.Is(err, mealbot.ErrPayerDoesNotExist) || errors.Is(err, mealbot.ErrRecipientDoesNotExist) {
			rw.WriteHeader(http.StatusBadRequest)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusOK)
}
