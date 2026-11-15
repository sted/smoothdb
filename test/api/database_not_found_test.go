package test_api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// A request to a database that does not exist, or that AllowedDatabases does
// not name, answers 404 with a message of its own. The cause of the failed
// activation (a pgx error, which can name the user and the host of a failed
// connection) goes only to the details of a verbose error body.
func TestDatabaseNotFoundMessage(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:8082/api/no_such_db/t", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, body %s", resp.StatusCode, body)
	}
	var e struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	if want := `database "no_such_db" not found`; e.Message != want {
		t.Errorf("message %q, expected %q", e.Message, want)
	}
}
