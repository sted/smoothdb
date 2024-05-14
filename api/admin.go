package api

import (
	"context"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

func InitAdminRouter(apiHelper Helper) {

	dbe := apiHelper.GetDBE()
	router := apiHelper.GetRouter()

	adminURL := apiHelper.BaseAdminURL()
	admin_db := router.Group(adminURL, apiHelper.MiddlewareStd())
	admin_dbe := router.Group(adminURL, apiHelper.MiddlewareDBE())
	admin_nodb := router.Group(adminURL)

	// ROLES

	roles := admin_db.Group("/roles")

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
		err := r.ReadJSON(&roleInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		role, err := database.CreateRole(c, &roleInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, role)
		} else {
			return WriteServerError(w, err)
		}
	})

	roles.Handle("PATCH", "/:rolename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var roleUpdate database.RoleUpdate
		name := r.Param("rolename")
		err := r.ReadJSON(&roleUpdate)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		err = database.UpdateRole(c, name, &roleUpdate)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, "")
		} else {
			return WriteServerError(w, err)
		}
	})

	roles.Handle("DELETE", "/:rolename", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("rolename")
		err := database.DeleteRole(c, name)
		if err == nil {
			return heligo.WriteHeader(w, http.StatusNoContent)
		} else {
			return WriteServerError(w, err)
		}
	})

	// USERS

	users := admin_db.Group("/users")

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
		err := r.ReadJSON(&userInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
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
			return heligo.WriteHeader(w, http.StatusNoContent)
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
			if targetType == "tables" {
				targetType = "table"
			}
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
		err := r.ReadJSON(&privilegeInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
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
		err := r.ReadJSON(&privilegeInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		err = database.DeletePrivilege(c, &privilegeInput)
		if err == nil {
			return heligo.WriteHeader(w, http.StatusOK)
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
		err := r.ReadJSON(&databaseInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		database, err := dbe.CreateDatabase(c, databaseInput.Name, databaseInput.Owner, false)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, database)
		} else {
			return WriteServerError(w, err)
		}
	})

	dbegroup.Handle("PATCH", "/:dbname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var databaseUpdate database.DatabaseUpdate
		name := r.Param("dbname")
		err := r.ReadJSON(&databaseUpdate)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		err = dbe.UpdateDatabase(c, name, &databaseUpdate)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, "")
		} else {
			return WriteServerError(w, err)
		}
	})

	dbegroup.Handle("DELETE", "/:dbname", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("dbname")

		err := dbe.DeleteDatabase(c, name)
		if err == nil {
			return heligo.WriteHeader(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases := admin_db.Group("/databases")

	// Alternatives routes for grants

	databases.Handle("GET", "/:dbname/grants", grantsGetHandler)
	// here :targettypes will be "tables"
	databases.Handle("GET", "/:dbname/:targettype/:targetname/grants", grantsGetHandler)

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
		err := r.ReadJSON(&schemaInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
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
			return heligo.WriteHeader(w, http.StatusNoContent)
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
		views, err := database.GetViews(c)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, views)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("GET", "/:dbname/views/:view", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("view")

		view, err := database.GetView(c, name)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, view)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("POST", "/:dbname/views", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var viewInput database.View
		err := r.ReadJSON(&viewInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		view, err := database.CreateView(c, &viewInput)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusCreated, view)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("DELETE", "/:dbname/views/:view", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		name := r.Param("view")

		err := database.DeleteView(c, name)
		if err == nil {
			return heligo.WriteHeader(w, http.StatusOK)
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
		err := r.ReadJSON(&columnInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
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
		table := r.Param("table")
		name := r.Param("column")
		err := r.ReadJSON(&columnUpdate)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		err = database.UpdateColumn(c, table, name, &columnUpdate)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, nil)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("DELETE", "/:dbname/tables/:table/columns/:column", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		table := r.Param("table")
		column := r.Param("column")

		err := database.DeleteColumn(c, table, column, false)
		if err == nil {
			return heligo.WriteHeader(w, http.StatusOK)
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
		err := r.ReadJSON(&constraintInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
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
			return heligo.WriteHeader(w, http.StatusOK)
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
		err := r.ReadJSON(&policyInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
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
			return heligo.WriteHeader(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	// FUNCTIONS

	databases.Handle("GET", "/:dbname/functions", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		functions, err := database.GetFunctions(c)
		if err == nil {
			return heligo.WriteJSON(w, http.StatusOK, functions)
		} else {
			return WriteServerError(w, err)
		}
	})

	databases.Handle("POST", "/:dbname/functions", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var functionInput database.Function
		err := r.ReadJSON(&functionInput)
		if err != nil {
			return WriteBadRequest(w, err)
		}
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
			return heligo.WriteHeader(w, http.StatusOK)
		} else {
			return WriteServerError(w, err)
		}
	})

	// SESSIONS

	admin_nodb.Handle("GET", "/sessions", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		stats := apiHelper.SessionStatistics()
		return heligo.WriteJSON(w, http.StatusOK, stats)
	})
}
