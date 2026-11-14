package database

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// maxTablesForFuzzySearch is PostgREST's maxDbTablesForFuzzySearch: a schema
// with as many tables gets no hint.
const maxTablesForFuzzySearch = 500

// TableNotFound returns PostgREST's TableNotFound error (PGRST205) when err
// is the undefined_table error (42P01) of the relation that the request
// names, and err otherwise. smoothdb runs the request even when the cache
// does not hold the relation, because the relation can be newer than the
// cache.
func TableNotFound(ctx context.Context, table string, err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42P01" {
		return err
	}
	gi := GetSmoothContext(ctx)
	schema, name := splitTableName(_s(table, gi.QueryOptions.Schema))
	if pgErr.Message != `relation "`+schema+"."+name+`" does not exist` {
		return err
	}
	return &RequestError{
		Status:  http.StatusNotFound,
		Code:    "PGRST205",
		Message: "Could not find the table '" + schema + "." + name + "' in the schema cache",
		Hint:    gi.Db.info.Load().tableNotFoundHint(schema, name),
	}
}

// tableNotFoundHint is PostgREST's tableNotFoundHint (Error.hs): the closest
// table of the schema.
func (si *SchemaInfo) tableNotFoundHint(schema, name string) string {
	if si == nil {
		return ""
	}
	var tables []string
	for qualified := range si.cachedTables {
		if s, t := splitTableName(qualified); s == schema {
			tables = append(tables, t)
		}
	}
	if len(tables) >= maxTablesForFuzzySearch {
		return ""
	}
	slices.Sort(tables)
	if found, ok := newFuzzySet(tables).getOne(name, 0.75); ok {
		return "Perhaps you meant the table '" + schema + "." + found + "'"
	}
	return ""
}

// functionIndex lists the functions of the database by name, for the
// PGRST202 error of a call that no function can take (see noRpc).
type functionIndex struct {
	// unnamed holds every qualified name of the database; it is true when an
	// overload has an unnamed input argument. smoothdb leaves a call to such
	// a name to PostgreSQL (card 32489 rpc-params).
	unnamed map[string]bool
	// callable holds, per schema, the names of the functions that PostgREST
	// keeps in its schema cache (SchemaCache.hs, funcsSqlQuery): every input
	// argument named, or one unnamed argument of a type that a body can fill.
	callable map[string][]string
}

// singleUnnamedTypes are the types of the single unnamed argument that
// PostgREST keeps in its schema cache.
var singleUnnamedTypes = []string{"bytea", "json", "jsonb", "text", "xml"}

// indexFunctions builds the functionIndex of the functions.
func indexFunctions(functions []Function) functionIndex {
	idx := functionIndex{unnamed: map[string]bool{}, callable: map[string][]string{}}
	for _, f := range functions {
		name := _s(f.Name, f.Schema)
		inputs := f.inputArguments()
		unnamed := 0
		for _, a := range inputs {
			if a.Name == "" {
				unnamed++
			}
		}
		idx.unnamed[name] = idx.unnamed[name] || unnamed > 0
		if unnamed == 0 || (unnamed == 1 && slices.Contains(singleUnnamedTypes, inputs[0].Type)) {
			idx.callable[f.Schema] = append(idx.callable[f.Schema], f.Name)
		}
	}
	return idx
}

// noRpc returns PostgREST's NoRpc error (PGRST202) when no function can take
// a call of name (schema.name) with the argument names keys, and nil when
// one can or when smoothdb leaves the call to PostgreSQL: a name with an
// overload that has an unnamed argument. post says the call is a POST, and
// contentType is the media type of its body.
func (si *SchemaInfo) noRpc(name string, keys []string, post bool, contentType string) error {
	if si == nil {
		return nil
	}
	unnamed, exists := si.functions.unnamed[name]
	if unnamed {
		return nil
	}
	overloads := si.cachedFunctions[name]
	if exists && len(overloads) == 0 {
		return nil // a function that the cache does not hold, a procedure
	}
	schema, proc := splitTableName(name)
	keys = slices.Sorted(slices.Values(keys))
	singleParam := post && (contentType == "text/plain" || contentType == "text/xml" || contentType == "application/octet-stream")

	params := " without parameters"
	if len(keys) > 0 {
		params = "(" + strings.Join(keys, ", ") + ")"
	}
	message := "Could not find the function " + schema + "." + proc
	if !singleParam {
		message += params
	}
	message += " in the schema cache"

	searched := " without parameters"
	if len(keys) == 1 {
		searched = " with parameter " + keys[0]
	} else if len(keys) > 1 {
		searched = " with parameters " + strings.Join(keys, ", ")
	}
	switch {
	case post && contentType == "text/plain":
		searched = " with a single unnamed text parameter"
	case post && contentType == "text/xml":
		searched = " with a single unnamed xml parameter"
	case post && contentType == "application/octet-stream":
		searched = " with a single unnamed bytea parameter"
	case post && contentType == "application/json":
		searched += " or with a single unnamed json/jsonb parameter"
	}

	err := &RequestError{
		Status:  http.StatusNotFound,
		Code:    "PGRST202",
		Message: message,
		Details: "Searched for the function " + schema + "." + proc + searched + ", but no matches were found in the schema cache.",
	}
	if !singleParam {
		err.Hint = si.noRpcHint(schema, proc, keys, overloads)
	}
	return err
}

// noRpcHint is PostgREST's noRpcHint (Error.hs): with no overload of the
// name, the closest function name of the schema; else the closest list of
// argument names among the overloads.
func (si *SchemaInfo) noRpcHint(schema, proc string, keys []string, overloads []Function) string {
	const hint = "Perhaps you meant to call the function "
	if len(overloads) == 0 {
		if found, ok := newFuzzySet(si.functions.callable[schema]).getOne(proc, 0.75); ok {
			return hint + schema + "." + found
		}
		return ""
	}
	listToText := func(names []string) string {
		return "(" + strings.Join(slices.Sorted(slices.Values(names)), ", ") + ")"
	}
	lists := make([]string, len(overloads))
	for i, f := range overloads {
		var names []string
		for _, a := range f.inputArguments() {
			names = append(names, a.Name)
		}
		lists[i] = listToText(names)
	}
	if found, ok := newFuzzySet(lists).getOne(listToText(keys), 0.33); ok {
		return hint + schema + "." + proc + found
	}
	return ""
}

// noRelBetweenHint is PostgREST's noRelBetweenHint (Error.hs): when origin
// has relationships in the schema, the closest relation it has to target;
// else the closest table with relationships to origin.
func (si *SchemaInfo) noRelBetweenHint(origin, target, schema string) string {
	if si == nil {
		return ""
	}
	var parents, children []string
	found := false
	for table, rels := range si.cachedRelationships {
		tschema, tname := splitTableName(table)
		if tschema != schema {
			continue
		}
		var related []string
		for _, rel := range rels {
			if rschema, rname := splitTableName(rel.RelatedTable); rschema == schema {
				related = append(related, rname)
			}
		}
		if len(related) == 0 {
			continue
		}
		parents = append(parents, tname)
		if tname == origin {
			found = true
			children = related
		}
	}
	if found {
		for _, m := range newFuzzySet(children).get(target, 0.33) {
			if m.score < 1 {
				return "Perhaps you meant '" + m.entry + "' instead of '" + target + "'."
			}
		}
		return ""
	}
	slices.Sort(parents)
	if p, ok := newFuzzySet(parents).getOne(origin, 0.33); ok {
		return "Perhaps you meant '" + p + "' instead of '" + origin + "'."
	}
	return ""
}
