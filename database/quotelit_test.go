package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// quoteLit writes the string literals of the JSON path keys, the full-text
// search configuration and the DDL comments. PostgreSQL reads the literal
// back as the original string whatever standard_conforming_strings says: with
// it off, a backslash in a plain '...' literal is an escape, so a key that
// ends in a backslash would escape the closing quote and the rest of the
// request would become SQL. The oracle is PostgreSQL: SELECT <literal> must
// return the input under both settings.
func TestQuoteLitReadsBackUnderBothStringModes(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	gi := GetSmoothContext(ctx)

	inputs := []string{
		"plain", // positive control: no quote, no backslash
		"it's",
		`back\slash`,
		`trailing\`,
		`\'`,
		`x\' or true --`,
		`\\`,
		"",
	}
	for _, mode := range []string{"on", "off"} {
		tx, err := gi.Conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, "SET LOCAL standard_conforming_strings = "+mode); err != nil {
			t.Fatal(err)
		}
		// no warning for each backslash in an E'' literal
		if _, err := tx.Exec(ctx, "SET LOCAL escape_string_warning = off"); err != nil {
			t.Fatal(err)
		}
		for _, in := range inputs {
			// a savepoint per input: a failed literal does not abort the others
			sp, err := tx.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var got string
			// an unnamed statement: a prepared statement cached under the
			// other setting would hide the difference
			err = sp.QueryRow(ctx, "SELECT "+quoteLit(in)+"::text", pgx.QueryExecModeExec).Scan(&got)
			_ = sp.Rollback(ctx)
			if err != nil {
				t.Errorf("standard_conforming_strings=%s, %q: %v", mode, in, err)
			} else if got != in {
				t.Errorf("standard_conforming_strings=%s: %q reads back as %q", mode, in, got)
			}
		}
		_ = tx.Rollback(ctx)
	}
}
