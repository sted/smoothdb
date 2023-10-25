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

func writeJSONContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"application/json; charset=utf-8"}
	}
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

func WriteJSON(w http.ResponseWriter, code int, obj any) (int, error) {
	writeJSONContentType(w)
	w.WriteHeader(code)
	if !bodyAllowedForStatus(code) {
		return code, nil
	}
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	_, err = w.Write(jsonBytes)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return code, err
}

func WriteJSONString(w http.ResponseWriter, code int, json []byte) (int, error) {
	writeJSONContentType(w)
	w.WriteHeader(code)
	if !bodyAllowedForStatus(code) {
		return code, nil
	}
	_, err := w.Write(json)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return code, err
}

func WriteEmpty(w http.ResponseWriter, code int) (int, error) {
	w.WriteHeader(code)
	return code, nil
}

func noRecordsForInsert(ctx context.Context, w http.ResponseWriter, records []database.Record) (bool, int) {
	if len(records) == 0 {
		gi := database.GetSmoothContext(ctx)
		var status int
		if gi.QueryOptions.ReturnRepresentation {
			status = http.StatusCreated
			WriteJSONString(w, status, []byte("[]"))
		} else {
			status = http.StatusNoContent
			WriteEmpty(w, status)
		}
		return true, status
	}
	return false, 0
}

func noRecordsForUpdate(ctx context.Context, w http.ResponseWriter, records []database.Record) (bool, int) {
	if len(records) == 0 || len(records[0]) == 0 {
		gi := database.GetSmoothContext(ctx)
		var status int
		if gi.QueryOptions.ReturnRepresentation {
			status = http.StatusOK
			WriteJSONString(w, status, []byte("[]"))
		} else {
			status = http.StatusNoContent
			WriteEmpty(w, status)
		}
		return true, status
	}
	return false, 0
}

func WriteError(w http.ResponseWriter, err error) (int, error) {
	switch err.(type) {
	case *database.ParseError, *database.BuildError:
		return WriteBadRequest(w, err)
	case *database.SerializeError:
		w.WriteHeader(http.StatusNotAcceptable)
		return http.StatusNotAcceptable, err
	default:
		return WriteServerError(w, err)
	}
}

func WriteBadRequest(w http.ResponseWriter, err error) (int, error) {
	return WriteJSON(w, http.StatusBadRequest, Data{"error": err.Error()})
}

func WriteServerError(w http.ResponseWriter, err error) (int, error) {
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
		return WriteJSON(w, status, Data{
			"code":    dberr.Code,
			"message": dberr.Message,
			"hint":    dberr.Hint,
		})
	} else if errors.Is(err, pgx.ErrNoRows) {
		return WriteJSON(w, http.StatusNotFound, nil)
	} else {
		return WriteJSON(w, http.StatusInternalServerError, Data{"error": err.Error()})
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
