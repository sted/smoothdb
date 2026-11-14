package api

import (
	"net/http/httptest"
	"strconv"
	"testing"
)

// ErrorVerbosity selects the shape of an error body:
//   - "verbose" writes every key with its value;
//   - "normal", the default, writes every key, but no hint or details of a
//     database error (see writeDatabaseError);
//   - "minimal" writes only code and message, as PostgREST 16.4 does with
//     client-error-verbosity = minimal (Error.hs errorPayload).
func TestErrorBodyVerbosity(t *testing.T) {
	savedVerbose, savedMinimal := verboseErrors, minimalErrors
	defer func() { verboseErrors, minimalErrors = savedVerbose, savedMinimal }()
	e := SmoothError{
		Code:    "PGRST205",
		Message: "Could not find the table 'test.itemsx' in the schema cache",
		Details: "some details",
		Hint:    "Perhaps you meant the table 'test.items'",
	}
	full := `{"code":"PGRST205","details":"some details","hint":"Perhaps you meant the table 'test.items'","message":"Could not find the table 'test.itemsx' in the schema cache"}`
	tests := []struct {
		mode string
		body string
	}{
		{"verbose", full},
		{"normal", full},
		{"minimal", `{"code":"PGRST205","message":"Could not find the table 'test.itemsx' in the schema cache"}`},
	}
	for _, tt := range tests {
		if err := SetErrorVerbosity(tt.mode); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		writeErrorBody(w, 404, e)
		if got := w.Body.String(); got != tt.body {
			t.Errorf("%s:\n want: %s\n  got: %s", tt.mode, tt.body, got)
		}
		if got := w.Header().Get("Content-Length"); got != strconv.Itoa(len(tt.body)) {
			t.Errorf("%s: Content-Length %s, want %d", tt.mode, got, len(tt.body))
		}
	}
	if err := SetErrorVerbosity("bogus"); err == nil {
		t.Error("SetErrorVerbosity(\"bogus\"): want an error")
	}
}
