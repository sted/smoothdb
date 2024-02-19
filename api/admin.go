package api

import (
	"context"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

func InitAdminRouter(apiHelper Helper) {

	dbe := apiHelper.GetDBE()
	router := apiHelper.Router()

	adminURL := apiHelper.BaseAdminURL()
	admin_dbe := router.Group(adminURL, apiHelper.MiddlewareDBE())
	admin_db := router.Group(adminURL, apiHelper.MiddlewareStd())
	admin_other := router.Group(adminURL)

	// ROLES

	roles := admin_dbe.Group("/roles")

	roles.Handle("GET", "", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		roles, err := database.GetRoles(c)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, roles)
		} else {
			return WriteServerError(w, err)
		}
	})

	roles.Handle("GET", "/:rolename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("rolename")
		role, err := database.GetRole(c, name)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, role)
		} else {
			return WriteServerError(w, err)
		}
	})

	roles.Handle("POST", "", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var roleInput database.Role
		r.ReadJSON(&roleInput)

		role, err := database.CreateRole(c, &roleInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, role)
		} else {
			return WriteServerError(w, err)
		}
	})

	roles.Handle("DELETE", "/:rolename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("rolename")

		err := database.DeleteRole(c, name)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusNoContent)
		} else {
			return WriteServerError(w, err)
		}
	})

	// USERS

	users := admin_dbe.Group("/users")

	users.Handle("GET", "", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		users, err := database.GetUsers(c)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, users)
		} else {
			return WriteServerError(w, err)
		}
	})

	users.Handle("GET", "/:username", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("username")
		user, err := database.GetUser(c, name)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, user)
		} else {
			return WriteServerError(w, err)
		}
	})

	users.Handle("POST", "", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var userInput database.User
		r.ReadJSON(&userInput)
		user, err := database.CreateUser(c, &userInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, user)
		} else {
			return WriteServerError(w, err)
		}
	})

	users.Handle("DELETE", "/:username", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("username")
		err := database.DeleteUser(c, name)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusNoContent)
		} else {
			return WriteServerError(w, err)
		}
	})

	// GRANTS

	grants := admin_db.Group("/grants")

	grantsGetHandler := func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var privileges []database.Privilege
		var err error

		dbname := r.Param("dbname")
		targetType := r.Param("targettype")
		targetName := r.Param("targetname")

		if targetType == "" {
			privileges, err = database.GetDatabasePrivileges(c, dbname)
		} else {
			privileges, err = database.GetPrivileges(c, targetType, targetName)
		}

		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, privileges)
		} else {
			return WriteServerError(w, err)
		}
	}
	grants.Handle("GET", "", grantsGetHandler)
	grants.Handle("GET", "/:dbname", grantsGetHandler)
	grants.Handle("GET", "/:dbname/:targettype", grantsGetHandler)
	grants.Handle("GET", "/:dbname/:targettype/:targetname", grantsGetHandler)

	grantsPostHandler := func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
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
		r.ReadJSON(&privilegeInput)

		priv, err := database.CreatePrivilege(c, &privilegeInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, priv)
		} else {
			return WriteServerError(w, err)
		}
	}
	grants.Handle("POST", "", grantsPostHandler)
	grants.Handle("POST", "/:dbname", grantsPostHandler)
	grants.Handle("POST", "/:dbname/:targettype/:targetname", grantsPostHandler)

	grantsDeleteHandler := func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
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
		r.ReadJSON(&privilegeInput)

		err := database.DeletePrivilege(c, &privilegeInput)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	}
	grants.Handle("DELETE", "/:dbname", grantsDeleteHandler)
	grants.Handle("DELETE", "/:dbname/:targettype/:targetname", grantsDeleteHandler)

	// DATABASES

	dbegroup := admin_dbe.Group("/databases")

	dbegroup.Handle("GET", "", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		databases, err := dbe.GetDatabases(c)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, databases)
		} else {
			return WriteServerError(w, err)
		}
	})

	dbegroup.Handle("GET", "/:dbname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("dbname")

		db, err := dbe.GetDatabase(c, name)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, db)
		} else {
			return WriteServerError(w, err)
		}
	})

	dbegroup.Handle("POST", "", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var databaseInput database.Database
		r.ReadJSON(&databaseInput)

		database, err := dbe.CreateDatabase(c, databaseInput.Name, false)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, database)
		} else {
			return WriteServerError(w, err)
		}
	})

	dbegroup.Handle("DELETE", "/:dbname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("dbname")

		err := dbe.DeleteDatabase(c, name)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases := admin_db.Group("/databases")

	// SCHEMAS

	databases.Handle("GET", "/:dbname/schemas", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		schemas, err := database.GetSchemas(c)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, schemas)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("POST", "/:dbname/schemas", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var schemaInput database.Schema
		r.ReadJSON(&schemaInput)

		schema, err := database.CreateSchema(c, schemaInput.Name)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, schema)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("DELETE", "/:dbname/schemas/:schema", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("schema")

		err := database.DeleteSchema(c, name, true)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusNoContent)
		} else {
			return WriteServerError(w, err)
		}
	})

	// TABLES

	databases.Handle("GET", "/:dbname/tables", TableListHandler)
	databases.Handle("GET", "/:dbname/tables/:table", TableGetHandler)
	databases.Handle("POST", "/:dbname/tables", TableCreateHandler)
	databases.Handle("PATCH", "/:dbname/tables/:table", TableUpdateHandler)
	databases.Handle("DELETE", "/:dbname/tables/:table", TableDeleteHandler)

	// VIEWS

	databases.Handle("GET", "/:dbname/views", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)

		views, err := db.GetViews(c)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, views)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("GET", "/:dbname/views/:view", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		name := r.Param("view")

		view, err := db.GetView(c, name)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, view)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("POST", "/:dbname/views", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		var viewInput database.View
		r.ReadJSON(&viewInput)

		view, err := db.CreateView(c, &viewInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, view)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("DELETE", "/:dbname/views/:view", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)
		name := r.Param("view")

		err := db.DeleteView(c, name)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	// COLUMNS

	databases.Handle("GET", "/:dbname/tables/:table/columns", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		table := r.Param("table")

		columns, err := database.GetColumns(c, table)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, columns)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("POST", "/:dbname/tables/:table/columns", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var columnInput database.Column
		columnInput.Table = r.Param("table")
		r.ReadJSON(&columnInput)
		if columnInput.Type == "" {
			columnInput.Type = "text"
		}

		column, err := database.CreateColumn(c, &columnInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, column)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("PATCH", "/:dbname/tables/:table/columns/:column", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var columnUpdate database.ColumnUpdate
		columnUpdate.Table = r.Param("table")
		columnUpdate.Name = r.Param("column")
		r.ReadJSON(&columnUpdate)

		column, err := database.UpdateColumn(c, &columnUpdate)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, column)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("DELETE", "/:dbname/tables/:table/columns/:column", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		table := r.Param("table")
		column := r.Param("column")

		err := database.DeleteColumn(c, table, column, false)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	// CONSTRAINTS

	databases.Handle("GET", "/:dbname/tables/:table/constraints", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		table := r.Param("table")

		constraints, err := database.GetConstraints(c, table)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, constraints)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("POST", "/:dbname/tables/:table/constraints", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var constraintInput database.Constraint
		constraintInput.Table = r.Param("table")
		r.ReadJSON(&constraintInput)

		constant, err := database.CreateConstraint(c, &constraintInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, constant)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("DELETE", "/:dbname/tables/:table/constraints/:name", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		table := r.Param("table")
		name := r.Param("name")

		err := database.DeleteConstraint(c, table, name)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	// POLICIES

	databases.Handle("GET", "/:dbname/tables/:table/policies", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		table := r.Param("table")

		policies, err := database.GetPolicies(c, table)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, policies)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("POST", "/:dbname/tables/:table/policies", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var policyInput database.Policy
		policyInput.Table = r.Param("table")
		r.ReadJSON(&policyInput)

		policy, err := database.CreatePolicy(c, &policyInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, policy)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("DELETE", "/:dbname/tables/:table/policies/:name", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		table := r.Param("table")
		name := r.Param("name")

		err := database.DeletePolicy(c, table, name)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	// FUNCTIONS

	databases.Handle("GET", "/:dbname/functions", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(c)

		policies, err := db.GetFunctions(c)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, policies)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("POST", "/:dbname/functions", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var functionInput database.Function
		r.ReadJSON(&functionInput)

		policy, err := database.CreateFunction(c, &functionInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, policy)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("DELETE", "/:dbname/functions/:name", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("name")

		err := database.DeleteFunction(c, name)
		if err == nil {
			return heligo.WriteEmpty(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	// SESSIONS

	admin_other.Handle("GET", "/sessions", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		stats := apiHelper.SessionStatistics()
		return heligo.WriteJSON(w, http.StatusOK, stats)
	})
}
