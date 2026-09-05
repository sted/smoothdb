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
