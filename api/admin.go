package api

import (
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitAdminRouter(root *gin.RouterGroup, dbe *database.DbEngine, handlers ...gin.HandlerFunc) *gin.RouterGroup {

	admin := root.Group("/admin", handlers...)

	// ROLES

	roles := admin.Group("/roles")

	roles.GET("/", func(c *gin.Context) {
		roles, _ := dbe.GetRoles(c)
		c.JSON(http.StatusOK, roles)
	})

	roles.GET("/:rolename", func(c *gin.Context) {
		name := c.Param("rolename")
		db, err := dbe.GetRole(c, name)
		if err == nil {
			c.JSON(http.StatusOK, db)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	roles.POST("/", func(c *gin.Context) {
		var roleInput database.Role
		c.BindJSON(&roleInput)
		role, err := dbe.CreateRole(c, &roleInput)
		if err == nil {
			c.JSON(http.StatusCreated, role)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	roles.DELETE("/:rolename", func(c *gin.Context) {
		name := c.Param("rolename")
		err := dbe.DeleteRole(c, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	// USERS

	users := admin.Group("/users")

	users.GET("/", func(c *gin.Context) {
		users, _ := dbe.GetUsers(c)
		c.JSON(http.StatusOK, users)
	})

	users.GET("/:username", func(c *gin.Context) {
		name := c.Param("username")
		db, err := dbe.GetUser(c, name)
		if err == nil {
			c.JSON(http.StatusOK, db)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	users.POST("/", func(c *gin.Context) {
		var userInput database.User
		c.BindJSON(&userInput)
		role, err := dbe.CreateUser(c, &userInput)
		if err == nil {
			c.JSON(http.StatusCreated, role)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	users.DELETE("/:username", func(c *gin.Context) {
		name := c.Param("username")
		err := dbe.DeleteUser(c, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	// DATABASES

	databases := admin.Group("/databases")

	databases.GET("/", func(c *gin.Context) {
		databases, _ := dbe.GetDatabases(c)
		c.JSON(http.StatusOK, databases)
	})

	databases.GET("/:dbname", func(c *gin.Context) {
		name := c.Param("dbname")
		db, err := dbe.GetDatabase(c, name)
		if err == nil {
			c.JSON(http.StatusOK, db)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/", func(c *gin.Context) {
		var databaseInput database.Database
		c.BindJSON(&databaseInput)
		database, err := dbe.CreateDatabase(c, databaseInput.Name)
		if err == nil {
			c.JSON(http.StatusCreated, database)
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
		tables, err := db.GetTables(c)
		if err == nil {
			c.JSON(http.StatusOK, tables)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.GET("/:dbname/tables/:table", func(c *gin.Context) {
		db := database.GetDb(c)
		name := c.Param("table")
		table, err := db.GetTable(c, name)
		if err == nil {
			c.JSON(http.StatusOK, table)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/tables/", func(c *gin.Context) {
		db := database.GetDb(c)
		var tableInput database.Table
		c.BindJSON(&tableInput)
		table, err := db.CreateTable(c, &tableInput)
		if err == nil {
			c.JSON(http.StatusCreated, table)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.PATCH("/:dbname/tables/:table", func(c *gin.Context) {
		db := database.GetDb(c)
		var tableUpdate database.TableUpdate
		tableUpdate.Name = c.Param("table")
		c.BindJSON(&tableUpdate)
		table, err := db.UpdateTable(c, &tableUpdate)
		if err == nil {
			c.JSON(http.StatusOK, table)
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

	databases.GET("/:dbname/views/:view", func(c *gin.Context) {
		db := database.GetDb(c)
		name := c.Param("view")
		view, err := db.GetView(c, name)
		if err == nil {
			c.JSON(http.StatusOK, view)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/views/", func(c *gin.Context) {
		db := database.GetDb(c)
		var viewInput database.View
		c.BindJSON(&viewInput)
		view, err := db.CreateView(c, &viewInput)
		if err == nil {
			c.JSON(http.StatusCreated, view)
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
		var columnInput database.Column
		columnInput.Table = c.Param("table")
		c.BindJSON(&columnInput)
		if columnInput.Type == "" {
			columnInput.Type = "text"
		}
		column, err := db.CreateColumn(c, &columnInput)
		if err == nil {
			c.JSON(http.StatusCreated, column)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.PATCH("/:dbname/tables/:table/columns/:column", func(c *gin.Context) {
		db := database.GetDb(c)
		var columnUpdate database.ColumnUpdate
		columnUpdate.Table = c.Param("table")
		columnUpdate.Name = c.Param("column")
		c.BindJSON(&columnUpdate)
		column, err := db.UpdateColumn(c, &columnUpdate)
		if err == nil {
			c.JSON(http.StatusOK, column)
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

	// CONSTRAINTS

	databases.GET("/:dbname/tables/:table/constraints", func(c *gin.Context) {
		db := database.GetDb(c)
		table := c.Param("table")
		columns, err := db.GetConstraints(c, table)
		if err == nil {
			c.JSON(http.StatusOK, columns)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/tables/:table/constraints", func(c *gin.Context) {
		db := database.GetDb(c)
		var constraintInput database.Constraint
		constraintInput.Table = c.Param("table")
		c.BindJSON(&constraintInput)
		column, err := db.CreateConstraint(c, &constraintInput)
		if err == nil {
			c.JSON(http.StatusCreated, column)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/tables/:table/constraints/:name", func(c *gin.Context) {
		db := database.GetDb(c)
		table := c.Param("table")
		name := c.Param("name")
		err := db.DeleteConstraint(c, table, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	// POLICIES

	databases.GET("/:dbname/tables/:table/policies", func(c *gin.Context) {
		db := database.GetDb(c)
		table := c.Param("table")
		columns, err := db.GetPolicies(c, table)
		if err == nil {
			c.JSON(http.StatusOK, columns)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.POST("/:dbname/tables/:table/policies", func(c *gin.Context) {
		db := database.GetDb(c)
		var policyInput database.Policy
		policyInput.Table = c.Param("table")
		c.BindJSON(&policyInput)
		column, err := db.CreatePolicy(c, &policyInput)
		if err == nil {
			c.JSON(http.StatusCreated, column)
		} else {
			prepareInternalServerError(c, err)
		}
	})

	databases.DELETE("/:dbname/tables/:table/policies/:name", func(c *gin.Context) {
		db := database.GetDb(c)
		table := c.Param("table")
		name := c.Param("name")
		err := db.DeletePolicy(c, table, name)
		if err != nil {
			prepareInternalServerError(c, err)
		}
	})

	return admin
}
