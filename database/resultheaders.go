package database

import (
	"context"
	"net/http"
	"strconv"
)

func SetResponseHeaders(ctx context.Context, w http.ResponseWriter, r *http.Request, count int64) {
	sc := GetSmoothContext(ctx)
	options := sc.QueryOptions
	// @@ must check if the table has a pk
	w.Header().Set("Content-Location", r.RequestURI)
	// Content-Range
	var rangeString string
	if count == 0 {
		rangeString = "*/*"
	} else {
		rangeString = strconv.FormatInt(options.RangeMin, 10) + "-" +
			strconv.FormatInt(options.RangeMin+count-1, 10) + "/*"
	}
	w.Header().Set("Content-Range", rangeString)
}
