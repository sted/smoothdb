package api

import (
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitSourcesRouter(root *gin.RouterGroup, handlers ...gin.HandlerFunc) *gin.RouterGroup {

	databases := root.Group("/", handlers...)

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
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/:sourcename", func(c *gin.Context) {
		db := database.GetDb(c)
		sourcename := c.Param("sourcename")
		records, err := prepareInputRecords(c)
		if err != nil {
			prepareInternalServerError(c, err)
		}
		data, count, err := db.CreateRecords(c, sourcename, records)
		if err == nil {
			if data == nil {
				c.JSON(http.StatusCreated, count)
			} else {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.String(http.StatusCreated, string(data))
			}
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.PATCH("/:dbname/:sourcename", func(c *gin.Context) {
		db := database.GetDb(c)
		sourcename := c.Param("sourcename")
		records, err := prepareInputRecords(c)
		if err != nil {
			prepareInternalServerError(c, err)
		}
		data, count, err := db.UpdateRecords(c, sourcename, records[0], c.Request.URL.Query())
		if err == nil {
			if data == nil {
				c.JSON(http.StatusOK, count)
			} else {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.String(http.StatusOK, string(data))
			}
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/:sourcename", func(c *gin.Context) {
		db := database.GetDb(c)
		sourcename := c.Param("sourcename")
		data, count, err := db.DeleteRecords(c, sourcename, c.Request.URL.Query())
		if err == nil {
			if data == nil {
				c.JSON(http.StatusOK, count)
			} else {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.String(http.StatusOK, string(data))
			}
		} else {
			prepareInternalServerError(c, err)
		}
	})

	return databases
}
