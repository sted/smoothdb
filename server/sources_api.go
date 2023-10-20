package server

import (
	"context"
	"heligo"
	"net/http"

	"github.com/smoothdb/smoothdb/database"
)

func (s *Server) initSourcesRouter() {

	api := s.GetRouter().Group(s.Config.BaseAPIURL, DatabaseMiddleware(s, false))

	// RECORDS

	api.Handle("GET", "/:dbname/:sourcename", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		json, err := db.GetRecords(c, sourcename, r.URL.Query())
		if err == nil {
			JSONString(w, http.StatusOK, json)
		} else {
			WriteError(w, err)
			return err
		}
		return nil
	})

	api.Handle("POST", "/:dbname/:sourcename", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		records, err := ReadInputRecords(r)
		if err != nil {
			WriteBadRequest(w, err)
			return err
		}
		// [] as input cause no inserts
		if noRecordsForInsert(c, w, records) {
			return nil
		}
		data, count, err := db.CreateRecords(c, sourcename, records, r.URL.Query())
		if err == nil {
			if data == nil {
				JSON(w, http.StatusCreated, count)
			} else {
				JSONString(w, http.StatusCreated, data)
			}
		} else {
			WriteError(w, err)
			return err
		}
		return nil
	})

	api.Handle("PATCH", "/:dbname/:sourcename", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		records, err := ReadInputRecords(r)
		if err != nil || len(records) > 1 {
			WriteBadRequest(w, err)
			return err
		}
		// {}, [] and [{}] as input cause no updates
		if noRecordsForUpdate(c, w, records) {
			return nil
		}
		data, _, err := db.UpdateRecords(c, sourcename, records[0], r.URL.Query())
		if err == nil {
			if data == nil {
				w.WriteHeader(http.StatusNoContent)
			} else {
				JSONString(w, http.StatusOK, data)
			}
		} else {
			WriteError(w, err)
			return err
		}
		return nil
	})

	api.Handle("DELETE", "/:dbname/:sourcename", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		sourcename := r.Param("sourcename")
		data, _, err := db.DeleteRecords(c, sourcename, r.URL.Query())
		if err == nil {
			if data == nil {
				w.WriteHeader(http.StatusNoContent)
			} else {
				JSONString(w, http.StatusOK, data)
			}
		} else {
			WriteError(w, err)
			return err
		}
		return nil
	})

	// FUNCTIONS

	api.Handle("GET", "/:dbname/rpc/:fname", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		fname := r.Param("fname")
		json, _, err := db.ExecFunction(c, fname, nil, r.URL.Query())
		if err == nil {
			JSONString(w, http.StatusOK, json)
		} else {
			WriteError(w, err)
		}

		return nil
	})

	api.Handle("POST", "/:dbname/rpc/:fname", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		fname := r.Param("fname")
		records, err := ReadInputRecords(r)
		if err != nil {
			WriteBadRequest(w, err)
			return err
		}
		// [] as input cause no inserts
		if noRecordsForInsert(c, w, records) {
			return nil
		}
		data, count, err := db.ExecFunction(c, fname, records[0], r.URL.Query())
		if err == nil {
			if data == nil {
				JSON(w, http.StatusOK, count)
			} else {
				JSONString(w, http.StatusOK, data)
			}
		} else {
			WriteServerError(w, err)
		}
		return nil
	})
}
