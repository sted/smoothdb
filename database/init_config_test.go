package database

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const initConfigTestURL = "postgresql://postgres:postgres@localhost:5432/postgres"

// captureCheckOutput redirects the startup checks' messages to a buffer.
func captureCheckOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := checkOutput
	checkOutput = &buf
	t.Cleanup(func() { checkOutput = old })
	return &buf
}

// execAdmin runs a statement on a connection of its own.
func execAdmin(t *testing.T, sql string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, initConfigTestURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatal(err)
	}
}

// backendsGone reports whether, within a second, no backend is left with
// the given application_name.
func backendsGone(t *testing.T, appName string) (bool, int) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, initConfigTestURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var n int
	for deadline := time.Now().Add(time.Second); ; {
		err := conn.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE application_name = $1", appName).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 || time.Now().After(deadline) {
			return n == 0, n
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// CheckDatabase checked the authenticator role twice and never the
// anonymous one: a privileged anonymous role started without a warning.
func TestCheckDatabaseChecksTheAnonRole(t *testing.T) {
	execAdmin(t, "DROP ROLE IF EXISTS smoothdb_check_anon")
	execAdmin(t, "CREATE ROLE smoothdb_check_anon NOLOGIN CREATEDB")
	t.Cleanup(func() { execAdmin(t, "DROP ROLE IF EXISTS smoothdb_check_anon") })
	out := captureCheckOutput(t)

	ok, err := CheckDatabase(&Config{URL: initConfigTestURL, AnonRole: "smoothdb_check_anon"})
	if err != nil || !ok {
		t.Fatalf("expected the check to pass with warnings, got %v %v", ok, err)
	}
	// control: the authenticator (postgres, a superuser) is warned about
	if !strings.Contains(out.String(), "authenticator role should have minimal privileges") {
		t.Fatalf("control: expected the authenticator warning, got %q", out)
	}
	if !strings.Contains(out.String(), "anonymous role should have minimal privileges") {
		t.Errorf("expected a warning on the privileged anonymous role, got %q", out)
	}
}

// A missing anonymous role was a nil dereference: GetRole's error was
// ignored and its nil result read.
func TestCheckDatabaseMissingAnonRole(t *testing.T) {
	out := captureCheckOutput(t)
	var ok bool
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("CheckDatabase panicked: %v", r)
			}
		}()
		ok, err = CheckDatabase(&Config{URL: initConfigTestURL, AnonRole: "smoothdb_no_such_anon"})
	}()
	if err != nil || !ok {
		t.Fatalf("a missing anonymous role is a warning, got %v %v", ok, err)
	}
	if !strings.Contains(out.String(), `"smoothdb_no_such_anon" does not exist`) {
		t.Errorf("expected a warning on the missing anonymous role, got %q", out)
	}
	// the same for the role checks called directly, on a missing role
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, initConfigTestURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	ctx = ContextWithDbConn(ctx, nil, conn)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("checkAuthRole panicked: %v", r)
			}
		}()
		if checkAuthRole(ctx, "smoothdb_no_such_auth") {
			t.Error("a missing authenticator role must stop the startup")
		}
	}()
}

// The connections opened by CheckDatabase and PrepareDatabase were never
// closed: each startup left a backend behind.
func TestStartupChecksCloseTheirConnection(t *testing.T) {
	captureCheckOutput(t)
	appName := fmt.Sprintf("smoothdb_check_%d", time.Now().UnixNano())
	url := initConfigTestURL + "?application_name=" + appName
	if _, err := CheckDatabase(&Config{URL: url}); err != nil {
		t.Fatal(err)
	}
	if gone, n := backendsGone(t, appName); !gone {
		t.Errorf("CheckDatabase left %d connection(s) open", n)
	}

	// PrepareDatabase, on its early error path: an unparsable database URL
	appName += "_prepare"
	url = initConfigTestURL + "?application_name=" + appName
	if err := PrepareDatabase(url, &Config{URL: "postgresql://localhost:notaport/x"}); err == nil {
		t.Fatal("expected an error for the unparsable URL")
	}
	if gone, n := backendsGone(t, appName); !gone {
		t.Errorf("PrepareDatabase left %d connection(s) open", n)
	}
}
