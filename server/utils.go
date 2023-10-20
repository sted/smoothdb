package server

import (
	"context"
	"encoding/json"
	"errors"
	"heligo"
	"io"
	"net/http"

	"github.com/smoothdb/smoothdb/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Data map[string]any

func writeJSONContentType(w heligo.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"application/json; charset=utf-8"}
	}
}

func writeJSON(w heligo.ResponseWriter, obj any) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonBytes)
	return err
}

// bodyAllowedForStatus is a copy of http.bodyAllowedForStatus non-exported function.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == http.StatusNoContent:
		return false
	case status == http.StatusNotModified:
		return false
	}
	return true
}

func JSON(w heligo.ResponseWriter, code int, obj any) error {
	writeJSONContentType(w)
	w.WriteHeader(code)
	if !bodyAllowedForStatus(code) {
		return nil
	}
	return writeJSON(w, obj)
}

func JSONString(w heligo.ResponseWriter, code int, json []byte) error {
	writeJSONContentType(w)
	w.WriteHeader(code)
	if !bodyAllowedForStatus(code) {
		return nil
	}
	_, err := w.Write(json)
	return err
}

func noRecordsForInsert(ctx context.Context, w heligo.ResponseWriter, records []database.Record) bool {
	if len(records) == 0 {
		gi := database.GetSmoothContext(ctx)
		if gi.QueryOptions.ReturnRepresentation {
			JSONString(w, http.StatusCreated, []byte("[]"))
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		return true
	}
	return false
}

func noRecordsForUpdate(ctx context.Context, w heligo.ResponseWriter, records []database.Record) bool {
	if len(records) == 0 || len(records[0]) == 0 {
		gi := database.GetSmoothContext(ctx)
		if gi.QueryOptions.ReturnRepresentation {
			JSONString(w, http.StatusOK, []byte("[]"))
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		return true
	}
	return false
}

func WriteError(w heligo.ResponseWriter, err error) {
	switch err.(type) {
	case *database.ParseError, *database.BuildError:
		WriteBadRequest(w, err)
	case *database.SerializeError:
		w.WriteHeader(http.StatusNotAcceptable)
	default:
		WriteServerError(w, err)
	}
}

func WriteBadRequest(w heligo.ResponseWriter, err error) {
	JSON(w, http.StatusBadRequest, Data{"error": err.Error()})
}

func WriteServerError(w heligo.ResponseWriter, err error) {
	if _, ok := err.(*pgconn.PgError); ok {
		dberr := err.(*pgconn.PgError)
		var status int
		switch dberr.Code {
		case "42501":
			status = http.StatusUnauthorized
		case "42P01", // undefined_table
			"42883": // undefined_function
			status = http.StatusNotFound
		case "42P04", // duplicate database
			"42P06", // duplicate schema
			"42P07", // duplicate table
			"23505": // unique constraint violation
			status = http.StatusConflict
		case "22P02", // invalid_text_representation
			"42703": // undefined_column
			status = http.StatusBadRequest
		default:
			status = http.StatusInternalServerError
		}
		JSON(w, status, Data{
			"code":    dberr.Code,
			"message": dberr.Message,
			"hint":    dberr.Hint,
		})
	} else if errors.Is(err, pgx.ErrNoRows) {
		JSON(w, http.StatusNotFound, nil)
	} else {
		JSON(w, http.StatusInternalServerError, Data{"error": err.Error()})
	}
}

// func (r *Request) Bind(obj any) error {
// 	decoder := json.NewDecoder(r.Request.Body)
// 	// if EnableDecoderUseNumber {
// 	// 	decoder.UseNumber()
// 	// }
// 	// if EnableDecoderDisallowUnknownFields {
// 	// 	decoder.DisallowUnknownFields()
// 	// }
// 	if err := decoder.Decode(obj); err != nil {
// 		return err
// 	}
// 	//return validate(obj)
// 	return nil
// }

func ReadInputRecords(r heligo.Request) ([]database.Record, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	isArray := false
	for _, c := range body {
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			continue
		}
		isArray = c == '['
		break
	}
	var records []database.Record
	if isArray {
		err = json.Unmarshal(body, &records)
		if err != nil {
			return nil, err
		}
	} else {
		var record database.Record
		err = json.Unmarshal(body, &record)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}
