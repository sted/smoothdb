package server

import (
	"context"
	"net/http"

	"github.com/smoothdb/smoothdb/database"
)

func InitSourcesRouter(router *Router, baseAPIURL string) {

	api := router.Group(baseAPIURL)

	// RECORDS

	api.HandleWithDb("GET", "/:dbname/:sourcename", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		json, err := db.GetRecords(c, sourcename, r.URL.Query())
		if err == nil {
			w.JSONString(http.StatusOK, json)
		} else {
			w.WriteError(err)
		}
	})

	api.HandleWithDb("POST", "/:dbname/:sourcename", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		records, err := r.ReadInputRecords()
		if err != nil {
			w.WriteBadRequest(err)
			return
		}
		// [] as input cause no inserts
		if noRecordsForInsert(c, w, records) {
			return
		}
		data, count, err := db.CreateRecords(c, sourcename, records, r.URL.Query())
		if err == nil {
			if data == nil {
				w.JSON(http.StatusCreated, count)
			} else {
				w.JSONString(http.StatusCreated, data)
			}
		} else {
			w.WriteError(err)
		}
	})

	api.HandleWithDb("PATCH", "/:dbname/:sourcename", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		records, err := r.ReadInputRecords()
		if err != nil || len(records) > 1 {
			w.WriteBadRequest(err)
			return
		}
		// {}, [] and [{}] as input cause no updates
		if noRecordsForUpdate(c, w, records) {
			return
		}
		data, _, err := db.UpdateRecords(c, sourcename, records[0], r.URL.Query())
		if err == nil {
			if data == nil {
				w.Status(http.StatusNoContent)
			} else {
				w.JSONString(http.StatusOK, data)
			}
		} else {
			w.WriteError(err)
		}
	})

	api.HandleWithDb("DELETE", "/:dbname/:sourcename", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		data, _, err := db.DeleteRecords(c, sourcename, r.URL.Query())
		if err == nil {
			if data == nil {
				w.Status(http.StatusNoContent)
			} else {
				w.JSONString(http.StatusOK, data)
			}
		} else {
			w.WriteError(err)
		}
	})

	// FUNCTIONS

	api.HandleWithDb("GET", "/:dbname/rpc/:fname", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		fname := r.Param("fname")
		json, _, err := db.ExecFunction(c, fname, nil, r.URL.Query())
		if err == nil {
			w.JSONString(http.StatusOK, json)
		} else {
			w.WriteError(err)
		}
	})

	api.HandleWithDb("POST", "/:dbname/rpc/:fname", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		fname := r.Param("fname")
		records, err := r.ReadInputRecords()
		if err != nil {
			w.WriteBadRequest(err)
			return
		}
		// [] as input cause no inserts
		if noRecordsForInsert(c, w, records) {
			return
		}
		data, count, err := db.ExecFunction(c, fname, records[0], r.URL.Query())
		if err == nil {
			if data == nil {
				w.JSON(http.StatusOK, count)
			} else {
				w.JSONString(http.StatusOK, data)
			}
		} else {
			w.WriteServerError(err)
		}
	})
}
