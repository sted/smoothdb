package api

import (
	"encoding/json"
	"fmt"
	"green/green-ds/database"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func prepareInputRecords(c *gin.Context) ([]database.Record, error) {

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	var m any
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}
	var records []database.Record
	records, ok := m.([]database.Record)
	if !ok {
		r, ok := m.(database.Record)
		if !ok {
			return nil, fmt.Errorf("invalid JSON in body request")
		}
		records = append(records, r)
	}
	return records, nil
}

func prepareInternalServerError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"description": err.Error()})
}
