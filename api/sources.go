package api

import (
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

func InitSourcesRouter(root *gin.RouterGroup, handlers ...gin.HandlerFunc) *gin.RouterGroup {

	databases := root.Group("/databases", handlers...)

	// RECORDS

	databases.GET("/:dbname/:sourcename", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		sourcename := c.Param("sourcename")
		fields, err := db.GetRecords(ctx, sourcename)
		if err == nil {
			c.JSON(http.StatusOK, fields)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/:sourcename", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		sourcename := c.Param("sourcename")
		record, err := preparePostRecord(c)
		if err != nil {
			prepareInternalServerError(c, err)
		}
		source, err := db.CreateRecord(ctx, sourcename, record)
		if err == nil {
			c.Render(http.StatusCreated, render.String{Format: string(source)})
		} else {
			prepareInternalServerError(c, err)
		}
	})
	return databases
}
