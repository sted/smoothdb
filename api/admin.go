package api

import (
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitAdminRouter(root *gin.RouterGroup, dbe *database.DBEngine, handlers ...gin.HandlerFunc) *gin.RouterGroup {

	admin := root.Group("/admin", handlers...)

	// DATABASES

	databases := admin.Group("/databases")

	databases.GET("/", func(c *gin.Context) {
		databases, _ := dbe.GetDatabases(c)
		c.JSON(http.StatusOK, databases)
	})

	databases.POST("/", func(c *gin.Context) {
		var database database.Database
		c.BindJSON(&database)
		db, err := dbe.CreateDatabase(c, database.Name)
		if err == nil {
			c.JSON(http.StatusCreated, db)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname", func(c *gin.Context) {
		name := c.Param("dbname")
		err := dbe.DeleteDatabase(c, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	// TABLES

	databases.GET("/:dbname/tables", func(c *gin.Context) {
		db := database.GetDb(c)
		sources, err := db.GetTables(c)
		if err == nil {
			c.JSON(http.StatusOK, sources)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/tables/", func(c *gin.Context) {
		db := database.GetDb(c)
		var table database.Table
		c.BindJSON(&table)
		source, err := db.CreateTable(c, &table)
		if err == nil {
			c.JSON(http.StatusCreated, source)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/tables/:table", func(c *gin.Context) {
		db := database.GetDb(c)
		name := c.Param("table")
		err := db.DeleteTable(c, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	// VIEWS

	databases.GET("/:dbname/views", func(c *gin.Context) {
		db := database.GetDb(c)
		views, err := db.GetViews(c)
		if err == nil {
			c.JSON(http.StatusOK, views)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/views/", func(c *gin.Context) {
		db := database.GetDb(c)
		var view database.View
		c.BindJSON(&view)
		v, err := db.CreateView(c, &view)
		if err == nil {
			c.JSON(http.StatusCreated, v)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/views/:view", func(c *gin.Context) {
		db := database.GetDb(c)
		name := c.Param("view")
		err := db.DeleteView(c, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	// COLUMNS

	databases.GET("/:dbname/tables/:table/columns", func(c *gin.Context) {
		db := database.GetDb(c)
		table := c.Param("table")
		columns, err := db.GetColumns(c, table)
		if err == nil {
			c.JSON(http.StatusOK, columns)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/tables/:table/columns", func(c *gin.Context) {
		db := database.GetDb(c)
		var column database.Column
		column.Table = c.Param("table")
		c.BindJSON(&column)
		if column.Type == "" {
			column.Type = "text"
		}
		source, err := db.CreateColumn(c, &column)
		if err == nil {
			c.JSON(http.StatusCreated, source)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/tables/:table/columns/:column", func(c *gin.Context) {
		db := database.GetDb(c)
		table := c.Param("table")
		column := c.Param("column")
		err := db.DeleteColumn(c, table, column, false)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})
	return admin
}
