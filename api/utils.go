package api

import (
	"encoding/json"
	"green/green-ds/database"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func preparePostRecord(c *gin.Context) (*database.Record, error) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	var m database.Record
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func prepareInternalServerError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"description": err.Error()})
}
