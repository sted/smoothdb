package api

import (
	"encoding/json"
	"green/green-ds/database"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

func prepareInputRecords(c *gin.Context) ([]database.Record, error) {

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	isArray := false
	for _, c := range body {
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			continue
		}
		isArray = c == '['
		break
	}
	var records []database.Record
	if isArray {
		err = json.Unmarshal(body, &records)
		if err != nil {
			return nil, err
		}
	} else {
		var record database.Record
		err = json.Unmarshal(body, &record)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func prepareBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"description": err.Error()})
}

func prepareServerError(c *gin.Context, err error) {
	if _, ok := err.(*pgconn.PgError); ok {
		dberr := err.(*pgconn.PgError)
		var status int
		switch dberr.Code {
		case "42501":
			status = http.StatusUnauthorized
		default:
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{
			"code":    dberr.Code,
			"message": dberr.Message,
			"hint":    dberr.Hint,
		})

	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	}
}
