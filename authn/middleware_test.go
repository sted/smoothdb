package authn

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// RequestMaxBytes is documented as "0 for unlimited", but MaxBytesReader(0)
// fails on the first byte, so 0 used to block every request body. A
// non-positive limit must leave the body untouched; a positive one still caps it.
func TestLimitBodyZeroMeansUnlimited(t *testing.T) {
	body := func() io.ReadCloser { return io.NopCloser(strings.NewReader("0123456789")) }

	got, err := io.ReadAll(LimitBody(httptest.NewRecorder(), body(), 0))
	if err != nil || string(got) != "0123456789" {
		t.Fatalf("limit 0: expected the whole body, got %q, %v", got, err)
	}
	got, err = io.ReadAll(LimitBody(httptest.NewRecorder(), body(), -1))
	if err != nil || string(got) != "0123456789" {
		t.Fatalf("limit -1: expected the whole body, got %q, %v", got, err)
	}

	// positive control: a positive limit still caps the body
	_, err = io.ReadAll(LimitBody(httptest.NewRecorder(), body(), 4))
	var maxErr *http.MaxBytesError
	if !errors.As(err, &maxErr) {
		t.Fatalf("limit 4: expected *http.MaxBytesError, got %v", err)
	}
}

// heldResponse sends nothing until send: the status, the headers and the body
// wait for the end of the transaction. Every Write copies, as io.Writer
// requires: a handler that reuses its slice after Write does not change the
// held body. discard drops the headers the handler set and keeps the ones set
// before it.
func TestHeldResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Server", "smoothdb")
	hr := holdResponse(rec)
	hr.Header().Set("Content-Range", "*/*")
	hr.WriteHeader(http.StatusCreated)
	body := []byte(`[{"id":1}]`)
	hr.Write(body)
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 || rec.Flushed {
		t.Fatalf("held response reached the writer: %d %q", rec.Code, rec.Body.String())
	}
	body[2] = 'X' // a handler that reuses its buffer after a single Write
	hr.send()
	if rec.Code != http.StatusCreated || rec.Body.String() != `[{"id":1}]` || rec.Header().Get("Content-Range") != "*/*" {
		t.Errorf("sent: %d %q %v", rec.Code, rec.Body.String(), rec.Header())
	}
	hr.release()

	rec = httptest.NewRecorder()
	rec.Header().Set("Server", "smoothdb")
	hr = holdResponse(rec)
	hr.Header().Set("Content-Location", "/x")
	first := []byte("ab")
	hr.Write(first)
	hr.Write([]byte("cd"))
	first[0] = 'X' // a writer that reuses its buffer after the second Write
	if string(hr.body) != "abcd" || hr.status != http.StatusOK {
		t.Errorf("two writes: %q %d, want \"abcd\" 200", hr.body, hr.status)
	}
	hr.discard()
	if rec.Header().Get("Content-Location") != "" || rec.Header().Get("Server") != "smoothdb" {
		t.Errorf("discard: headers %v, want Server only", rec.Header())
	}
	hr.release()
}
