package api

import (
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitSourcesRouter(root *gin.RouterGroup, handlers ...gin.HandlerFunc) *gin.RouterGroup {

	databases := root.Group("/api", handlers...)

	// RECORDS

	databases.GET("/:dbname/:sourcename", func(c *gin.Context) {
		db := database.GetDb(c)
		sourcename := c.Param("sourcename")
		json, err := db.GetRecords(c, sourcename, c.Request.URL.Query())
		if err == nil {
			c.Writer.Header().Set("Content-Type", "application/json")
			c.String(http.StatusOK, string(json))
		} else if _, ok := err.(*database.ParseError); ok {
			prepareBadRequest(c, err)
		} else {
			prepareServerError(c, err)
		}
	})

	databases.POST("/:dbname/:sourcename", func(c *gin.Context) {
		db := database.GetDb(c)
		sourcename := c.Param("sourcename")
		records, err := prepareInputRecords(c)
		if err != nil {
			prepareBadRequest(c, err)
			return
		}
		// [] as input cause no inserts
		if noRecordsForInsert(c, records) {
			return
		}
		data, count, err := db.CreateRecords(c, sourcename, records, c.Request.URL.Query())
		if err == nil {
			if data == nil {
				c.JSON(http.StatusCreated, count)
			} else {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.String(http.StatusCreated, string(data))
			}
		} else {
			prepareServerError(c, err)
		}
	})

	databases.PATCH("/:dbname/:sourcename", func(c *gin.Context) {
		db := database.GetDb(c)
		sourcename := c.Param("sourcename")
		records, err := prepareInputRecords(c)
		if err != nil || len(records) > 1 {
			prepareBadRequest(c, err)
			return
		}
		// {}, [] and [{}] as input cause no updates
		if noRecordsForUpdate(c, records) {
			return
		}
		data, _, err := db.UpdateRecords(c, sourcename, records[0], c.Request.URL.Query())
		if err == nil {
			if data == nil {
				c.Status(http.StatusNoContent)
			} else {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.String(http.StatusOK, string(data))
			}
		} else {
			prepareServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/:sourcename", func(c *gin.Context) {
		db := database.GetDb(c)
		sourcename := c.Param("sourcename")
		data, _, err := db.DeleteRecords(c, sourcename, c.Request.URL.Query())
		if err == nil {
			if data == nil {
				c.Status(http.StatusNoContent)
			} else {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.String(http.StatusOK, string(data))
			}
		} else {
			prepareServerError(c, err)
		}
	})

	return databases
}
