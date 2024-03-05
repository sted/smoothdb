package api

import (
	"context"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

func TableListHandler(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
	tables, err := database.GetTables(c)
	if err == nil {
		return heligo.WriteJSON(w, http.StatusOK, tables)
	} else {
		return WriteServerError(w, err)
	}
}

func TableCreateHandler(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
	var tableInput database.Table
	err := r.ReadJSON(&tableInput)
	if err != nil {
		return WriteBadRequest(w, err)
	}
	table, err := database.CreateTable(c, &tableInput)
	if err == nil {
		if table != nil {
			return heligo.WriteJSON(w, http.StatusCreated, table)
		} else {
			return heligo.WriteEmpty(w, http.StatusCreated)
		}
	} else {
		return WriteServerError(w, err)
	}
}

func TableGetHandler(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
	name := r.Param("table")
	table, err := database.GetTable(c, name)
	if err == nil {
		return heligo.WriteJSON(w, http.StatusOK, table)
	} else {
		return WriteServerError(w, err)
	}
}

func TableUpdateHandler(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
	var tableUpdate database.TableUpdate
	tableUpdate.Name = r.Param("table")
	err := r.ReadJSON(&tableUpdate)
	if err != nil {
		return WriteBadRequest(w, err)
	}
	table, err := database.UpdateTable(c, &tableUpdate)
	if err == nil {
		if table != nil {
			return heligo.WriteJSON(w, http.StatusCreated, table)
		} else {
			return heligo.WriteEmpty(w, http.StatusCreated)
		}
	} else {
		return WriteServerError(w, err)
	}
}

func TableDeleteHandler(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
	name := r.Param("table")
	err := database.DeleteTable(c, name, false)
	if err == nil {
		return heligo.WriteEmpty(w, http.StatusNoContent)
	} else {
		return WriteServerError(w, err)
	}
}
