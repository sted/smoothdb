package api

import (
	"context"
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitAdminRouter(root *gin.RouterGroup, dbe *database.DBEngine, handlers ...gin.HandlerFunc) *gin.RouterGroup {

	admin := root.Group("/admin", handlers...)

	// DATABASES

	databases := admin.Group("/databases")

	databases.GET("/", func(c *gin.Context) {
		ctx := context.Background()
		databases := dbe.GetDatabases(ctx)
		c.JSON(http.StatusOK, databases)
	})

	databases.POST("/:dbname", func(c *gin.Context) {
		ctx := context.Background()
		name := c.Param("dbname")
		db, err := dbe.CreateDatabase(ctx, name)
		if err == nil {
			c.JSON(http.StatusCreated, db)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname", func(c *gin.Context) {
		ctx := context.Background()
		name := c.Param("dbname")
		err := dbe.DeleteDatabase(ctx, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	// SOURCES

	databases.GET("/:dbname/sources", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		sources, err := db.GetSources(ctx)
		if err == nil {
			c.JSON(http.StatusOK, sources)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/sources/:sourcename", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		name := c.Param("sourcename")
		source, err := db.CreateSource(ctx, name)
		if err == nil {
			c.JSON(http.StatusCreated, source)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/sources/:sourcename", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		name := c.Param("sourcename")
		err := db.DeleteSource(ctx, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	// FIELDS

	databases.GET("/:dbname/sources/:sourcename/fields", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		sourcename := c.Param("sourcename")
		fields, err := db.GetFields(ctx, sourcename)
		if err == nil {
			c.JSON(http.StatusOK, fields)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/sources/:sourcename/fields/:fieldname", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		var field database.Field
		field.Source = c.Param("sourcename")
		field.Name = c.Param("fieldname")
		c.BindJSON(&field)
		if field.Type == "" {
			field.Type = "text"
		}
		source, err := db.CreateField(ctx, &field)
		if err == nil {
			c.JSON(http.StatusCreated, source)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/sources/:sourcename/fields/:fieldname", func(c *gin.Context) {
		ctx := database.NewContext(c)
		db := database.GetDb(ctx)
		sourcename := c.Param("sourcename")
		fieldname := c.Param("fieldname")
		err := db.DeleteField(ctx, sourcename, fieldname)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})
	return admin
}
