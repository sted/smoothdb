package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/smoothdb/smoothdb/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Data map[string]any

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

type ResponseWriter interface {
	http.ResponseWriter

	Status() int
	Err() error

	JSON(code int, obj any) error
	JSONString(code int, json []byte) error
	WriteError(error)
	WriteBadRequest(error)
	WriteServerError(error)
}

type responseWriter struct {
	http.ResponseWriter
	status int
	err    error
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Err() error {
	return w.err
}

// Status sets the HTTP response code.
func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) writeJSONContentType() {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"application/json; charset=utf-8"}
	}
}

func (w *responseWriter) writeJSON(obj any) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonBytes)
	return err
}

func (w *responseWriter) JSON(code int, obj any) error {
	w.writeJSONContentType()
	w.WriteHeader(code)
	if !bodyAllowedForStatus(code) {
		return nil
	}
	return w.writeJSON(obj)
}

func (w *responseWriter) JSONString(code int, json []byte) error {
	w.writeJSONContentType()
	w.WriteHeader(code)
	if !bodyAllowedForStatus(code) {
		return nil
	}
	_, err := w.Write(json)
	return err
}

func noRecordsForInsert(ctx context.Context, w ResponseWriter, records []database.Record) bool {
	if len(records) == 0 {
		gi := database.GetSmoothContext(ctx)
		if gi.QueryOptions.ReturnRepresentation {
			w.JSONString(http.StatusCreated, []byte("[]"))
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		return true
	}
	return false
}

func noRecordsForUpdate(ctx context.Context, w ResponseWriter, records []database.Record) bool {
	if len(records) == 0 || len(records[0]) == 0 {
		gi := database.GetSmoothContext(ctx)
		if gi.QueryOptions.ReturnRepresentation {
			w.JSONString(http.StatusOK, []byte("[]"))
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		return true
	}
	return false
}

func (w *responseWriter) WriteError(err error) {
	switch err.(type) {
	case *database.ParseError, *database.BuildError:
		w.WriteBadRequest(err)
	case *database.SerializeError:
		w.WriteHeader(http.StatusNotAcceptable)
	default:
		w.WriteServerError(err)
	}
}

func (w *responseWriter) WriteBadRequest(err error) {
	w.err = err
	w.JSON(http.StatusBadRequest, Data{"error": err.Error()})
}

func (w *responseWriter) WriteServerError(err error) {
	w.err = err
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
		w.JSON(status, Data{
			"code":    dberr.Code,
			"message": dberr.Message,
			"hint":    dberr.Hint,
		})
	} else if errors.Is(err, pgx.ErrNoRows) {
		w.JSON(http.StatusNotFound, nil)
	} else {
		w.JSON(http.StatusInternalServerError, Data{"error": err.Error()})
	}
}

type Request struct {
	*http.Request
	params Params
}

func (r *Request) Param(name string) string {
	return r.params.ByName(name)
}

func (r *Request) Bind(obj any) error {
	decoder := json.NewDecoder(r.Response.Body)
	// if EnableDecoderUseNumber {
	// 	decoder.UseNumber()
	// }
	// if EnableDecoderDisallowUnknownFields {
	// 	decoder.DisallowUnknownFields()
	// }
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	//return validate(obj)
	return nil
}

func (r *Request) ReadInputRecords() ([]database.Record, error) {
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
