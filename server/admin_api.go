package server

import (
	"context"
	"net/http"

	"github.com/smoothdb/smoothdb/database"
)

func TableListHandler(c context.Context, w ResponseWriter, r *Request) {
	db := database.GetDb(c)
	tables, err := db.GetTables(c)
	if err == nil {
		w.JSON(http.StatusOK, tables)
	} else {
		w.WriteServerError(err)
	}
}

func TableCreateHandler(c context.Context, w ResponseWriter, r *Request) {
	db := database.GetDb(c)
	var tableInput database.Table
	r.Bind(&tableInput)
	table, err := db.CreateTable(c, &tableInput)
	if err == nil {
		w.JSON(http.StatusCreated, table)
	} else {
		w.WriteServerError(err)
	}
}

func InitAdminRouter(root *Router, dbe *database.DbEngine, baseAdminURL string) {

	admin := root.Group(baseAdminURL)

	// ROLES

	roles := admin.Group("/roles")

	roles.HandleWithDb("GET", "/", func(c context.Context, w ResponseWriter, r *Request) {
		roles, _ := dbe.GetRoles(c)
		w.JSON(http.StatusOK, roles)
	})

	roles.HandleWithDb("GET", "/:rolename", func(c context.Context, w ResponseWriter, r *Request) {
		name := r.Param("rolename")
		role, err := dbe.GetRole(c, name)
		if err == nil {
			w.JSON(http.StatusOK, role)
		} else {
			w.WriteServerError(err)
		}
	})

	roles.HandleWithDb("POST", "/", func(c context.Context, w ResponseWriter, r *Request) {
		var roleInput database.Role
		r.Bind(&roleInput)

		role, err := dbe.CreateRole(c, &roleInput)
		if err == nil {
			w.JSON(http.StatusCreated, role)
		} else {
			w.WriteServerError(err)
		}
	})

	roles.HandleWithDb("DELETE", "/:rolename", func(c context.Context, w ResponseWriter, r *Request) {
		name := r.Param("rolename")

		err := dbe.DeleteRole(c, name)
		if err == nil {
			w.Status(http.StatusNoContent)
		} else {
			w.WriteServerError(err)
		}
	})

	// USERS

	users := admin.Group("/users")

	users.HandleWithDb("GET", "/", func(c context.Context, w ResponseWriter, r *Request) {
		users, _ := dbe.GetUsers(c)
		w.JSON(http.StatusOK, users)
	})

	users.HandleWithDb("GET", "/:username", func(c context.Context, w ResponseWriter, r *Request) {
		name := r.Param("username")
		user, err := dbe.GetUser(c, name)
		if err == nil {
			w.JSON(http.StatusOK, user)
		} else {
			w.WriteServerError(err)
		}
	})

	users.HandleWithDb("POST", "/", func(c context.Context, w ResponseWriter, r *Request) {
		var userInput database.User
		r.Bind(&userInput)
		user, err := dbe.CreateUser(c, &userInput)
		if err == nil {
			w.JSON(http.StatusCreated, user)
		} else {
			w.WriteServerError(err)
		}
	})

	users.HandleWithDb("DELETE", "/:username", func(c context.Context, w ResponseWriter, r *Request) {
		name := r.Param("username")
		err := dbe.DeleteUser(c, name)
		if err == nil {
			w.Status(http.StatusNoContent)
		} else {
			w.WriteServerError(err)
		}
	})

	// GRANTS

	grants := admin.Group("/grants")

	grantsGetHandler := func(c context.Context, w ResponseWriter, r *Request) {
		var privileges []database.Privilege
		var err error

		dbname := r.Param("dbname")
		targetType := r.Param("targettype")
		targetName := r.Param("targetname")

		if targetType == "" {
			privileges, err = dbe.GetDatabasePrivileges(c, dbname)
		} else {
			db := database.GetDb(c)
			privileges, err = db.GetPrivileges(c, targetType, targetName)
		}

		if err == nil {
			w.JSON(http.StatusOK, privileges)
		} else {
			w.WriteServerError(err)
		}
	}
	grants.HandleWithDb("GET", "/", grantsGetHandler)
	grants.HandleWithDb("GET", "/:dbname", grantsGetHandler)
	grants.HandleWithDb("GET", "/:dbname/:targettype", grantsGetHandler)
	grants.HandleWithDb("GET", "/:dbname/:targettype/:targetname", grantsGetHandler)

	grantsPostHandler := func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)

		dbname := r.Param("dbname")
		targetType := r.Param("targettype")
		targetName := r.Param("targetname")

		var privilegeInput database.Privilege
		if dbname != "" {
			if targetType == "" {
				targetType = "database"
				targetName = dbname
			}

			privilegeInput.TargetType = targetType
			privilegeInput.TargetName = targetName
		}
		r.Bind(&privilegeInput)

		priv, err := db.CreatePrivilege(c, &privilegeInput)
		if err == nil {
			w.JSON(http.StatusCreated, priv)
		} else {
			w.WriteServerError(err)
		}
	}
	grants.HandleWithDb("POST", "/", grantsPostHandler)
	grants.HandleWithDb("POST", "/:dbname", grantsPostHandler)
	grants.HandleWithDb("POST", "/:dbname/:targettype/:targetname", grantsPostHandler)

	grantsDeleteHandler := func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)

		dbname := r.Param("dbname")
		targetType := r.Param("targettype")
		targetName := r.Param("targetname")

		if targetType == "" {
			targetType = "database"
			targetName = dbname
		}

		var privilegeInput database.Privilege
		privilegeInput.TargetType = targetType
		privilegeInput.TargetName = targetName
		r.Bind(&privilegeInput)

		err := db.DeletePrivilege(c, &privilegeInput)
		if err != nil {
			w.WriteServerError(err)
		}
	}
	grants.HandleWithDb("DELETE", "/:dbname", grantsDeleteHandler)
	grants.HandleWithDb("DELETE", "/:dbname/:targettype/:targetname", grantsDeleteHandler)

	// DATABASES

	// A group using DBE instead of a specific db
	dbgroup := root.Group(baseAdminURL + "/databases")

	dbgroup.HandleWithDb("GET", "/", func(c context.Context, w ResponseWriter, r *Request) {
		databases, _ := dbe.GetDatabases(c)
		w.JSON(http.StatusOK, databases)
	})

	dbgroup.HandleWithDb("GET", "/:dbname", func(c context.Context, w ResponseWriter, r *Request) {
		name := r.Param("dbname")

		db, err := dbe.GetDatabase(c, name)
		if err == nil {
			w.JSON(http.StatusOK, db)
		} else {
			w.WriteServerError(err)
		}
	})

	dbgroup.HandleWithDb("POST", "/", func(c context.Context, w ResponseWriter, r *Request) {
		var databaseInput database.Database
		r.Bind(&databaseInput)

		database, err := dbe.CreateDatabase(c, databaseInput.Name)
		if err == nil {
			w.JSON(http.StatusCreated, database)
		} else {
			w.WriteServerError(err)
		}
	})

	dbgroup.HandleWithDb("DELETE", "/:dbname", func(c context.Context, w ResponseWriter, r *Request) {
		name := r.Param("dbname")

		err := dbe.DeleteDatabase(c, name)
		if err != nil {
			w.WriteServerError(err)
		}
	})

	databases := admin.Group("/databases")

	// SCHEMAS

	databases.HandleWithDb("GET", "/:dbname/schemas", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)

		schemas, err := db.GetSchemas(c)
		if err == nil {
			w.JSON(http.StatusOK, schemas)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("POST", "/:dbname/schemas/", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		var schemaInput database.Schema
		r.Bind(&schemaInput)

		schema, err := db.CreateSchema(c, schemaInput.Name)
		if err == nil {
			w.JSON(http.StatusCreated, schema)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("DELETE", "/:dbname/schemas/:schema", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		name := r.Param("schema")

		err := db.DeleteSchema(c, name, true)
		if err == nil {
			w.Status(http.StatusNoContent)
		} else {
			w.WriteServerError(err)
		}
	})

	// TABLES

	databases.HandleWithDb("GET", "/:dbname/tables", TableListHandler)

	databases.HandleWithDb("GET", "/:dbname/tables/:table", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		name := r.Param("table")

		table, err := db.GetTable(c, name)
		if err == nil {
			w.JSON(http.StatusOK, table)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("POST", "/:dbname/tables", TableCreateHandler)

	databases.HandleWithDb("PATCH", "/:dbname/tables/:table", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		var tableUpdate database.TableUpdate
		tableUpdate.Name = r.Param("table")
		r.Bind(&tableUpdate)

		table, err := db.UpdateTable(c, &tableUpdate)
		if err == nil {
			w.JSON(http.StatusOK, table)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("DELETE", "/:dbname/tables/:table", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		name := r.Param("table")

		err := db.DeleteTable(c, name)
		if err == nil {
			w.Status(http.StatusNoContent)
		} else {
			w.WriteServerError(err)
		}
	})

	// VIEWS

	databases.HandleWithDb("GET", "/:dbname/views", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)

		views, err := db.GetViews(c)
		if err == nil {
			w.JSON(http.StatusOK, views)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("GET", "/:dbname/views/:view", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		name := r.Param("view")

		view, err := db.GetView(c, name)
		if err == nil {
			w.JSON(http.StatusOK, view)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("POST", "/:dbname/views/", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		var viewInput database.View
		r.Bind(&viewInput)

		view, err := db.CreateView(c, &viewInput)
		if err == nil {
			w.JSON(http.StatusCreated, view)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("DELETE", "/:dbname/views/:view", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		name := r.Param("view")

		err := db.DeleteView(c, name)
		if err != nil {
			w.WriteServerError(err)
		}
	})

	// COLUMNS

	databases.HandleWithDb("GET", "/:dbname/tables/:table/columns", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		table := r.Param("table")

		columns, err := db.GetColumns(c, table)
		if err == nil {
			w.JSON(http.StatusOK, columns)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("POST", "/:dbname/tables/:table/columns", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		var columnInput database.Column
		columnInput.Table = r.Param("table")
		r.Bind(&columnInput)
		if columnInput.Type == "" {
			columnInput.Type = "text"
		}

		column, err := db.CreateColumn(c, &columnInput)
		if err == nil {
			w.JSON(http.StatusCreated, column)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("PATCH", "/:dbname/tables/:table/columns/:column", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		var columnUpdate database.ColumnUpdate
		columnUpdate.Table = r.Param("table")
		columnUpdate.Name = r.Param("column")
		r.Bind(&columnUpdate)

		column, err := db.UpdateColumn(c, &columnUpdate)
		if err == nil {
			w.JSON(http.StatusOK, column)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("DELETE", "/:dbname/tables/:table/columns/:column", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		table := r.Param("table")
		column := r.Param("column")

		err := db.DeleteColumn(c, table, column, false)
		if err != nil {
			w.WriteServerError(err)
		}
	})

	// CONSTRAINTS

	databases.HandleWithDb("GET", "/:dbname/tables/:table/constraints", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		table := r.Param("table")

		constraints, err := db.GetConstraints(c, table)
		if err == nil {
			w.JSON(http.StatusOK, constraints)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("POST", "/:dbname/tables/:table/constraints", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		var constraintInput database.Constraint
		constraintInput.Table = r.Param("table")
		r.Bind(&constraintInput)

		constant, err := db.CreateConstraint(c, &constraintInput)
		if err == nil {
			w.JSON(http.StatusCreated, constant)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("DELETE", "/:dbname/tables/:table/constraints/:name", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		table := r.Param("table")
		name := r.Param("name")

		err := db.DeleteConstraint(c, table, name)
		if err != nil {
			w.WriteServerError(err)
		}
	})

	// POLICIES

	databases.HandleWithDb("GET", "/:dbname/tables/:table/policies", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		table := r.Param("table")

		policies, err := db.GetPolicies(c, table)
		if err == nil {
			w.JSON(http.StatusOK, policies)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("POST", "/:dbname/tables/:table/policies", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		var policyInput database.Policy
		policyInput.Table = r.Param("table")
		r.Bind(&policyInput)

		policy, err := db.CreatePolicy(c, &policyInput)
		if err == nil {
			w.JSON(http.StatusCreated, policy)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("DELETE", "/:dbname/tables/:table/policies/:name", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		table := r.Param("table")
		name := r.Param("name")

		err := db.DeletePolicy(c, table, name)
		if err != nil {
			w.WriteServerError(err)
		}
	})

	// FUNCTIONS

	databases.HandleWithDb("GET", "/:dbname/functions", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)

		policies, err := db.GetFunctions(c)
		if err == nil {
			w.JSON(http.StatusOK, policies)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("POST", "/:dbname/functions", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		var functionInput database.Function
		r.Bind(&functionInput)

		policy, err := db.CreateFunction(c, &functionInput)
		if err == nil {
			w.JSON(http.StatusCreated, policy)
		} else {
			w.WriteServerError(err)
		}
	})

	databases.HandleWithDb("DELETE", "/:dbname/functions/:name", func(c context.Context, w ResponseWriter, r *Request) {
		db := database.GetDb(c)
		name := r.Param("name")

		err := db.DeleteFunction(c, name)
		if err != nil {
			w.WriteServerError(err)
		}
	})
}
