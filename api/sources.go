package api

import (
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitSourcesRouter(root *gin.RouterGroup, handlers ...gin.HandlerFunc) *gin.RouterGroup {

	databases := root.Group("/databases", handlers...)

	// RECORDS

	databases.GET("/:dbname/:sourcename", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		sourcename := c.Param("sourcename")
		json, err := db.GetRecords(ctx, sourcename, c.Request.URL.Query())
		if err == nil {
			c.String(http.StatusOK, string(json))
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/:sourcename", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		sourcename := c.Param("sourcename")
		records, err := prepareInputRecords(c)
		if err != nil {
			prepareInternalServerError(c, err)
		}
		_, err = db.CreateRecords(ctx, sourcename, records)
		if err == nil {
			c.String(http.StatusCreated, "")
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/:sourcename", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		sourcename := c.Param("sourcename")
		_, err := db.DeleteRecords(ctx, sourcename, c.Request.URL.Query())
		if err == nil {
			c.String(http.StatusOK, "")
		} else {
			prepareInternalServerError(c, err)
		}
	})

	return databases
}
