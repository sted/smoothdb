package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_ColumnHintEmbed covers an embed hinted by a column,
// relation!column(...), when several relations carry the same foreign key
// column (card 34668 column-hint-embed). The fixtures expose private.books
// through the views books, forties_books, fifties_books and sixties_books,
// each with author_id; the hint selects a relationship of the relation named
// only (Plan.hs findRel). Bodies from PostgREST 16.4 on the fixtures.
func TestPostgREST_ColumnHintEmbed(t *testing.T) {

	tests := []test.Test{
		// @@ added
		{
			Description: "a column hint embeds the view named, with its filter",
			Query:       "/authors?select=name,books:sixties_books!author_id(title)&id=eq.1",
			Expected:    `[{"name":"George Orwell","books":[]}]`,
			Status:      200,
		},
		{
			Description: "a column hint embeds another view of the same table",
			Query:       "/authors?select=name,books:forties_books!author_id(title)&id=eq.1",
			Expected:    `[{"name":"George Orwell","books":[{"title":"1984"}]}]`,
			Status:      200,
		},
		{
			Description: "the column of the origin works as a hint too",
			Query:       "/authors?select=name,books:sixties_books!id(title)&id=eq.1",
			Expected:    `[{"name":"George Orwell","books":[]}]`,
			Status:      200,
		},
		{
			// PostgREST answers 400 PGRST200; smoothdb's error body differs
			Description: "a column hint does not reach a relation that is not named",
			Query:       "/authors?select=name,books:nosuch!author_id(title)&id=eq.1",
			Status:      400,
		},
		{
			// EmbedDisambiguationSpec: in a self-relationship the column hint
			// selects the one-to-many side, by the referencing column
			Description: "a column hint on a self-relationship embeds the children",
			Query:       "/organizations?select=name,auditees:organizations!auditor(name)&id=eq.2",
			Expected:    `[{"name":"Auditor Org","auditees":[{"name":"Acme"},{"name":"Umbrella"}]}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
