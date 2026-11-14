package database

import "testing"

// The Location of PostgREST 16.4 for POST /simple_pk {"k":"a b&c/d+e"} with
// return=headers-only is /simple_pk?k=eq.a%20b%26c%2Fd%2Be.
func TestAppendQueryEscape(t *testing.T) {
	cases := map[string]string{
		"a b&c/d+e":  "a%20b%26c%2Fd%2Be",
		"Enzo-2.0_~": "Enzo-2.0_~",
		"è":          "%C3%A8",
	}
	for in, want := range cases {
		if got := string(appendQueryEscape(nil, in)); got != want {
			t.Errorf("appendQueryEscape(%q) = %q, want %q", in, got, want)
		}
	}
}
