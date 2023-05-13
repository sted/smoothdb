package api

import (
	"net/http"

	"github.com/smoothdb/smoothdb/database"

	"github.com/gin-gonic/gin"
)

func prepareError(c *gin.Context, err error) {
	switch err.(type) {
	case *database.ParseError, *database.BuildError:
		prepareBadRequest(c, err)
	case *database.SerializeError:
		c.Status(http.StatusNotAcceptable)
	default:
		prepareServerError(c, err)
	}
}

func InitSourcesRouter(root *gin.RouterGroup, handlers ...gin.HandlerFunc) *gin.RouterGroup {

	api := root.Group("/api", handlers...)

	// RECORDS

	api.GET("/:dbname/:sourcename", func(c *gin.Context) {
		db := database.GetDb(c)
		sourcename := c.Param("sourcename")
		json, err := db.GetRecords(c, sourcename, c.Request.URL.Query())
		if err == nil {
			c.Writer.Header().Set("Content-Type", "application/json")
			c.String(http.StatusOK, string(json))
		} else {
			prepareError(c, err)
		}
	})

	api.POST("/:dbname/:sourcename", func(c *gin.Context) {
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
			prepareError(c, err)
		}
	})

	api.PATCH("/:dbname/:sourcename", func(c *gin.Context) {
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
			prepareError(c, err)
		}
	})

	api.DELETE("/:dbname/:sourcename", func(c *gin.Context) {
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
			prepareError(c, err)
		}
	})

	// FUNCTIONS

	api.POST("/:dbname/rpc/:fname", func(c *gin.Context) {
		db := database.GetDb(c)
		fname := c.Param("fname")
		records, err := prepareInputRecords(c)
		if err != nil {
			prepareBadRequest(c, err)
			return
		}
		// [] as input cause no inserts
		if noRecordsForInsert(c, records) {
			return
		}
		data, count, err := db.ExecFunction(c, fname, records[0], c.Request.URL.Query())
		if err == nil {
			if data == nil {
				c.JSON(http.StatusOK, count)
			} else {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.String(http.StatusOK, string(data))
			}
		} else {
			prepareServerError(c, err)
		}
	})

	return api
}
