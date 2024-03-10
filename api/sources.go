package api

import (
	"context"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

func InitSourcesRouter(apiHelper Helper) {

	baseURL := apiHelper.BaseAPIURL()
	if !apiHelper.HasShortAPIURL() {
		baseURL += "/:dbname"
	}
	api := apiHelper.Router().Group(baseURL, apiHelper.MiddlewareStd())

	// RECORDS

	api.Handle("GET", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		json, count, err := database.GetRecords(c, sourcename, r.URL.Query())
		if err == nil {
			database.SetResponseHeaders(c, w, r.Request, count)
			return heligo.WriteJSONString(w, http.StatusOK, json)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		records, err := ReadInputRecords(r)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		// [] as input cause no inserts
		if ok, status := noRecordsForInsert(c, w, records); ok {
			return status, nil
		}
		data, count, err := database.CreateRecords(c, sourcename, records, r.URL.Query())
		if err == nil {
			if data == nil {
				return heligo.WriteJSON(w, http.StatusCreated, count)
			} else {
				return heligo.WriteJSONString(w, http.StatusCreated, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("PATCH", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		records, err := ReadInputRecords(r)
		if err != nil || len(records) > 1 {
			return WriteBadRequest(w, err)
		}
		// {}, [] and [{}] as input cause no updates
		if ok, status := noRecordsForUpdate(c, w, records); ok {
			return status, nil
		}
		data, _, err := database.UpdateRecords(c, sourcename, records[0], r.URL.Query())
		if err == nil {
			if data == nil {
				return heligo.WriteEmpty(w, http.StatusNoContent)
			} else {
				return heligo.WriteJSONString(w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("DELETE", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		data, _, err := database.DeleteRecords(c, sourcename, r.URL.Query())
		if err == nil {
			if data == nil {
				return heligo.WriteEmpty(w, http.StatusNoContent)
			} else {
				return heligo.WriteJSONString(w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	// FUNCTIONS

	api.Handle("GET", "/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		fname := r.Param("fname")
		json, count, err := database.ExecFunction(c, fname, nil, r.URL.Query(), true)
		if err == nil {
			database.SetResponseHeaders(c, w, r.Request, count)
			return heligo.WriteJSONString(w, http.StatusOK, json)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		fname := r.Param("fname")
		records, err := ReadInputRecords(r)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		// [] as input cause no inserts
		if ok, status := noRecordsForInsert(c, w, records); ok {
			return status, nil
		}
		data, count, err := database.ExecFunction(c, fname, records[0], r.URL.Query(), false)
		if err == nil {
			if data == nil {
				return heligo.WriteJSON(w, http.StatusOK, count)
			} else {
				database.SetResponseHeaders(c, w, r.Request, count)
				return heligo.WriteJSONString(w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})
}
