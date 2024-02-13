package database

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
