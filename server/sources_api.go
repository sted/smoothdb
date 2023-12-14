package server

import (
	"context"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

func (s *Server) initSourcesRouter() {

	api := s.GetRouter().Group(s.Config.BaseAPIURL, DatabaseMiddleware(s, false))

	// RECORDS

	api.Handle("GET", "/:dbname/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		json, err := db.GetRecords(c, sourcename, r.URL.Query())
		if err == nil {
			w.Header().Set("Content-Location", r.RequestURI)
			return WriteJSONString(w, http.StatusOK, json)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/:dbname/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		records, err := ReadInputRecords(r)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		// [] as input cause no inserts
		if ok, status := noRecordsForInsert(c, w, records); ok {
			return status, nil
		}
		data, count, err := db.CreateRecords(c, sourcename, records, r.URL.Query())
		if err == nil {
			if data == nil {
				return WriteJSON(w, http.StatusCreated, count)
			} else {
				return WriteJSONString(w, http.StatusCreated, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("PATCH", "/:dbname/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		records, err := ReadInputRecords(r)
		if err != nil || len(records) > 1 {
			return WriteBadRequest(w, err)
		}
		// {}, [] and [{}] as input cause no updates
		if ok, status := noRecordsForUpdate(c, w, records); ok {
			return status, nil
		}
		data, _, err := db.UpdateRecords(c, sourcename, records[0], r.URL.Query())
		if err == nil {
			if data == nil {
				return WriteEmpty(w, http.StatusNoContent)
			} else {
				return WriteJSONString(w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("DELETE", "/:dbname/:sourcename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		data, _, err := db.DeleteRecords(c, sourcename, r.URL.Query())
		if err == nil {
			if data == nil {
				return WriteEmpty(w, http.StatusNoContent)
			} else {
				return WriteJSONString(w, http.StatusOK, data)
			}
		} else {
			return WriteError(w, err)
		}
	})

	// FUNCTIONS

	api.Handle("GET", "/:dbname/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		fname := r.Param("fname")
		json, _, err := db.ExecFunction(c, fname, nil, r.URL.Query())
		if err == nil {
			return WriteJSONString(w, http.StatusOK, json)
		} else {
			return WriteError(w, err)
		}
	})

	api.Handle("POST", "/:dbname/rpc/:fname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		fname := r.Param("fname")
		records, err := ReadInputRecords(r)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		// [] as input cause no inserts
		if ok, status := noRecordsForInsert(c, w, records); ok {
			return status, nil
		}
		data, count, err := db.ExecFunction(c, fname, records[0], r.URL.Query())
		if err == nil {
			if data == nil {
				return WriteJSON(w, http.StatusOK, count)
			} else {
				return WriteJSONString(w, http.StatusOK, data)
			}
		} else {
			return WriteServerError(w, err)
		}
	})
}
