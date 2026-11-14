package database

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// RequestError is a refusal that smoothdb decides before the database runs
// the request, as PostgREST decides its ApiRequestError and SchemaCacheError
// (Error.hs). It carries the HTTP status, the PGRST code, and the message,
// details and hint of the error body. Details is a string or a JSON value;
// an empty one is written as null, as an empty Hint is.
type RequestError struct {
	Status  int
	Code    string
	Message string
	Details any
	Hint    string
}

func (e *RequestError) Error() string { return e.Message }

// notEmbedded is PostgREST's NotEmbedded (PGRST108): a filter, an order or a
// limit names a relation that the select does not embed.
func notEmbedded(resource string) *RequestError {
	return &RequestError{
		Status:  http.StatusBadRequest,
		Code:    "PGRST108",
		Message: "'" + resource + "' is not an embedded resource in this request",
		Hint:    "Verify that '" + resource + "' is included in the 'select' query parameter.",
	}
}

// relatedOrderNotToOne is PostgREST's RelatedOrderNotToOne (PGRST118): an
// order on an embed that is not to-one.
func relatedOrderNotToOne(origin, target string) *RequestError {
	return &RequestError{
		Status:  http.StatusBadRequest,
		Code:    "PGRST118",
		Message: "A related order on '" + target + "' is not possible",
		Details: "'" + origin + "' and '" + target + "' do not form a many-to-one or one-to-one relationship",
	}
}

// ambiguousRelBetween is PostgREST's AmbiguousRelBetween (PGRST201): more
// than one relationship joins origin and target. The details list the
// relationships in PostgREST's order (sortRelationships), and the hint
// names the embed hint of each one.
func ambiguousRelBetween(origin, target string, rels []Relationship) *RequestError {
	rels = slices.Clone(rels)
	sortRelationships(rels)
	details := make([]map[string]string, len(rels))
	hints := make([]string, len(rels))
	for i := range rels {
		details[i], hints[i] = describeRelationship(&rels[i])
	}
	return &RequestError{
		Status:  http.StatusMultipleChoices,
		Code:    "PGRST201",
		Message: "Could not embed because more than one relationship was found for '" + origin + "' and '" + target + "'",
		Details: details,
		Hint:    "Try changing '" + target + "' to one of the following: " + strings.Join(hints, ", ") + ". Find the desired relationship in the 'details' key.",
	}
}

// noRelBetween is PostgREST's NoRelBetween (PGRST200): no relationship
// joins origin and target, with the embed hint fk if the request gives one.
// qualifiedOrigin is the origin with its schema.
func noRelBetween(origin, target, fk, qualifiedOrigin string, info *SchemaInfo) *RequestError {
	schema, _ := splitTableName(qualifiedOrigin)
	using := ""
	if fk != "" {
		using = "' using the hint '" + fk
	}
	return &RequestError{
		Status:  http.StatusBadRequest,
		Code:    "PGRST200",
		Message: "Could not find a relationship between '" + origin + "' and '" + target + "' in the schema cache",
		Details: "Searched for a foreign key relationship between '" + origin + "' and '" + target + using + "' in the schema '" + schema + "', but no matches were found.",
		Hint:    info.noRelBetweenHint(origin, target, schema),
	}
}

// cardinalityNames are the names of the RelTypes in PostgREST's bodies.
var cardinalityNames = [...]string{
	O2M: "one-to-many",
	M2O: "many-to-one",
	O2O: "one-to-one",
	M2M: "many-to-many",
}

// describeRelationship returns the entry of a relationship in the details
// of PGRST201 (Error.hs, compressedRel) and its embed hint (relHint).
func describeRelationship(rel *Relationship) (map[string]string, string) {
	_, table := splitTableName(rel.Table)
	_, related := splitTableName(rel.RelatedTable)
	cols := func(c []string) string { return "(" + strings.Join(c, ", ") + ")" }
	var relationship, hint string
	if rel.Type == M2M {
		_, junction := splitTableName(rel.JunctionTable)
		relationship = junction + " using " + rel.JForeignKey + cols(rel.JColumns) +
			" and " + rel.JRelatedForeignKey + cols(rel.JRelatedColumns)
		hint = junction
	} else {
		relationship = rel.ForeignKey + " using " + table + cols(rel.Columns) +
			" and " + related + cols(rel.RelatedColumns)
		hint = rel.ForeignKey
	}
	entry := map[string]string{
		"cardinality":  cardinalityNames[rel.Type],
		"embedding":    table + " with " + related,
		"relationship": relationship,
	}
	return entry, "'" + related + "!" + hint + "'"
}

// sortRelationships puts relationships in the order of PostgREST's schema
// cache, which sorts them with the derived Ord of Relationship: by
// cardinality (O2M, M2O, O2O, M2M), then by constraint, then by columns.
// An M2M sorts by junction, then by its two constraints and its columns.
func sortRelationships(rels []Relationship) {
	key := func(r *Relationship) []string {
		if r.Type == M2M {
			_, junction := splitTableName(r.JunctionTable)
			return slices.Concat([]string{junction, r.JForeignKey, r.JRelatedForeignKey}, r.JColumns, r.JRelatedColumns)
		}
		cols := make([]string, 0, 2*len(r.Columns))
		for i := range r.Columns {
			cols = append(cols, r.Columns[i], r.RelatedColumns[i])
		}
		return append([]string{r.ForeignKey}, cols...)
	}
	slices.SortStableFunc(rels, func(a, b Relationship) int {
		if a.Type != b.Type {
			return int(a.Type) - int(b.Type)
		}
		return slices.Compare(key(&a), key(&b))
	})
}

// columnNotFound is PostgREST's ColumnNotFound (PGRST204): a column of the
// payload or of ?columns is not in the table.
func columnNotFound(table, column string) *RequestError {
	return &RequestError{
		Status:  http.StatusBadRequest,
		Code:    "PGRST204",
		Message: "Could not find the '" + column + "' column of '" + table + "' in the schema cache",
	}
}

// invalidBody is PostgREST's InvalidBody (PGRST102): the payload cannot be
// read as the records of the request.
func invalidBody(message string) *RequestError {
	return &RequestError{Status: http.StatusBadRequest, Code: "PGRST102", Message: message}
}

// InvalidBody returns PostgREST's InvalidBody error (PGRST102).
func InvalidBody(message string) *RequestError { return invalidBody(message) }

// negativeLimit is PostgREST's InvalidRange NegativeLimit (PGRST103): a
// negative limit makes the range of the request empty.
var negativeLimit = &RequestError{
	Status:  http.StatusRequestedRangeNotSatisfiable,
	Code:    "PGRST103",
	Message: "Requested range not satisfiable",
	Details: "Limit should be greater than or equal to zero.",
}

// lowerGTUpper is PostgREST's InvalidRange LowerGTUpper (PGRST103): the
// Range header has a lower boundary above its upper boundary.
var lowerGTUpper = &RequestError{
	Status:  http.StatusRequestedRangeNotSatisfiable,
	Code:    "PGRST103",
	Message: "Requested range not satisfiable",
	Details: "The lower boundary must be lower than or equal to the upper boundary in the Range header.",
}

// singularityError is PostgREST's SingularityError (PGRST116): a request
// for a single object reads or writes another number of rows.
func singularityError(rows int64) *RequestError {
	return &RequestError{
		Status:  http.StatusNotAcceptable,
		Code:    "PGRST116",
		Message: "Cannot coerce the result to a single JSON object",
		Details: "The result contains " + strconv.FormatInt(rows, 10) + " rows",
	}
}

// errOctetColumns is the binary serializer's refusal of more than one
// column. runRead answers it as mediaTypeError.
var errOctetColumns = &RequestError{Status: http.StatusNotAcceptable, Code: "PGRST107"}

// mediaTypeError is PostgREST's MediaTypeError (PGRST107), which names every
// alternative of the Accept headers.
func mediaTypeError(accept []string) *RequestError {
	return &RequestError{
		Status:  http.StatusNotAcceptable,
		Code:    "PGRST107",
		Message: "None of these media types are available: " + acceptedMediaTypes(accept),
	}
}

// aggregatesNotAllowed is PostgREST's AggregatesNotAllowed (PGRST123).
var aggregatesNotAllowed = &RequestError{
	Status:  http.StatusBadRequest,
	Code:    "PGRST123",
	Message: "Use of aggregate functions is not allowed",
}

// func ErrorToHTTPStatus(err error) (int, any) {
// 	if _, ok := err.(*pgconn.PgError); ok {
// 		dberr := err.(*pgconn.PgError)
// 		var status int
// 		switch dberr.Code {
// 		case "42501":
// 			status = http.StatusUnauthorized
// 		case "42P01", // undefined_table
// 			"42883": // undefined_function
// 			status = http.StatusNotFound
// 		case "42P04", // duplicate database
// 			"42P06", // duplicate schema
// 			"42P07", // duplicate table
// 			"23505": // unique constraint violation
// 			status = http.StatusConflict
// 		case "22P02", // invalid_text_representation
// 			"42703": // undefined_column
// 			status = http.StatusBadRequest
// 		default:
// 			status = http.StatusInternalServerError
// 		}
// 		return status, map[string]string{
// 			"code":    dberr.Code,
// 			"message": dberr.Message,
// 			"hint":    dberr.Hint,
// 		}
// 	} else if errors.Is(err, pgx.ErrNoRows) {
// 		return http.StatusNotFound, nil
// 	} else {
// 		return http.StatusInternalServerError, Data{"error": err.Error()}
// 	}
// }
