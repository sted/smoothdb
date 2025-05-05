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
	api := apiHelper.GetRouter().Group(baseURL, apiHelper.MiddlewareStd())

	// TABLES

	api.Handle("GET", "", TableListHandler)
	api.Handle("GET", "/$info/:table", TableGetHandler)

	// RECORDS

	api.Handle("GET", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		json, count, err := database.GetRecords(c, sourcename, r.URL.Query())
		if err == nil {
			status := SetResponseHeaders(c, w, r, count)
			if status >= http.StatusBadRequest {
				return heligo.WriteHeader(w, status)
			}
			if status == 0 {
				status = http.StatusOK
			}
			return WriteContent(c, w, status, json)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		records, status, err := ReadRequest(c, w, r)
		if err != nil || status != 0 {
			return status, err
		}
		data, count, err := database.CreateRecords(c, sourcename, records, r.URL.Query())
		if err == nil {
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				return heligo.WriteJSON(w, http.StatusCreated, count)
			} else {
				return WriteContent(c, w, http.StatusCreated, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("PATCH", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		records, status, err := ReadRequest(c, w, r)
		if err != nil || status != 0 {
			return status, err
		}
		data, count, err := database.UpdateRecords(c, sourcename, records[0], r.URL.Query())
		if err == nil {
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				return heligo.WriteHeader(w, http.StatusNoContent)
			} else {
				return WriteContent(c, w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("DELETE", "/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		sourcename := r.Param("sourcename")
		data, count, err := database.DeleteRecords(c, sourcename, r.URL.Query())
		if err == nil {
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				return heligo.WriteHeader(w, http.StatusNoContent)
			} else {
				return WriteContent(c, w, http.StatusOK, data)
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
			status := SetResponseHeaders(c, w, r, count)
			if status >= http.StatusBadRequest {
				return heligo.WriteHeader(w, status)
			}
			if status == 0 {
				status = http.StatusOK
			}
			return WriteContent(c, w, status, json)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		fname := r.Param("fname")
		records, status, err := ReadRequest(c, w, r)
		if err != nil || status != 0 {
			return status, err
		}
		data, count, err := database.ExecFunction(c, fname, records[0], r.URL.Query(), false)
		if err == nil {
			SetResponseHeaders(c, w, r, count)
			if data == nil {
				return heligo.WriteJSON(w, http.StatusOK, count)
			} else {
				return WriteContent(c, w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})
}
