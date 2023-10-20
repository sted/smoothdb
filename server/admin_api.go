package server

import (
	"context"
	"heligo"
	"net/http"

	"github.com/smoothdb/smoothdb/database"
)

func TableListHandler(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
	db := database.GetDb(c)
	tables, err := db.GetTables(c)
	if err == nil {
		JSON(w, http.StatusOK, tables)
	} else {
		WriteServerError(w, err)
		return err
	}
	return nil
}

func TableCreateHandler(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
	db := database.GetDb(c)
	var tableInput database.Table
	r.Bind(&tableInput)
	table, err := db.CreateTable(c, &tableInput)
	if err == nil {
		JSON(w, http.StatusCreated, table)
	} else {
		WriteServerError(w, err)
		return err
	}
	return nil
}

func (s *Server) initAdminRouter() {

	dbe := s.DBE
	router := s.GetRouter()

	admin_dbe := router.Group(s.Config.BaseAdminURL, DatabaseMiddleware(s, true))
	admin_db := router.Group(s.Config.BaseAdminURL, DatabaseMiddleware(s, false))

	// ROLES

	roles := admin_dbe.Group("/roles")

	roles.Handle("GET", "", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		roles, _ := dbe.GetRoles(c)
		JSON(w, http.StatusOK, roles)
		return nil
	})

	roles.Handle("GET", "/:rolename", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		name := r.Param("rolename")
		role, err := dbe.GetRole(c, name)
		if err == nil {
			JSON(w, http.StatusOK, role)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	roles.Handle("POST", "", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		var roleInput database.Role
		r.Bind(&roleInput)

		role, err := dbe.CreateRole(c, &roleInput)
		if err == nil {
			JSON(w, http.StatusCreated, role)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	roles.Handle("DELETE", "/:rolename", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		name := r.Param("rolename")

		err := dbe.DeleteRole(c, name)
		if err == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	// USERS

	users := admin_dbe.Group("/users")

	users.Handle("GET", "", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		users, _ := dbe.GetUsers(c)
		JSON(w, http.StatusOK, users)
		return nil
	})

	users.Handle("GET", "/:username", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		name := r.Param("username")
		user, err := dbe.GetUser(c, name)
		if err == nil {
			JSON(w, http.StatusOK, user)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	users.Handle("POST", "", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		var userInput database.User
		r.Bind(&userInput)
		user, err := dbe.CreateUser(c, &userInput)
		if err == nil {
			JSON(w, http.StatusCreated, user)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	users.Handle("DELETE", "/:username", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		name := r.Param("username")
		err := dbe.DeleteUser(c, name)
		if err == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	// GRANTS

	grants := admin_db.Group("/grants")

	grantsGetHandler := func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
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
			JSON(w, http.StatusOK, privileges)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	}
	grants.Handle("GET", "", grantsGetHandler)
	grants.Handle("GET", "/:dbname", grantsGetHandler)
	grants.Handle("GET", "/:dbname/:targettype", grantsGetHandler)
	grants.Handle("GET", "/:dbname/:targettype/:targetname", grantsGetHandler)

	grantsPostHandler := func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
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
			JSON(w, http.StatusCreated, priv)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	}
	grants.Handle("POST", "", grantsPostHandler)
	grants.Handle("POST", "/:dbname", grantsPostHandler)
	grants.Handle("POST", "/:dbname/:targettype/:targetname", grantsPostHandler)

	grantsDeleteHandler := func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
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
			WriteServerError(w, err)
			return err
		}
		return nil
	}
	grants.Handle("DELETE", "/:dbname", grantsDeleteHandler)
	grants.Handle("DELETE", "/:dbname/:targettype/:targetname", grantsDeleteHandler)

	// DATABASES

	dbegroup := admin_dbe.Group("/databases")

	dbegroup.Handle("GET", "", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		databases, _ := dbe.GetDatabases(c)
		JSON(w, http.StatusOK, databases)
		return nil
	})

	dbegroup.Handle("GET", "/:dbname", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		name := r.Param("dbname")

		db, err := dbe.GetDatabase(c, name)
		if err == nil {
			JSON(w, http.StatusOK, db)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	dbegroup.Handle("POST", "", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		var databaseInput database.Database
		r.Bind(&databaseInput)

		database, err := dbe.CreateDatabase(c, databaseInput.Name)
		if err == nil {
			JSON(w, http.StatusCreated, database)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	dbegroup.Handle("DELETE", "/:dbname", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		name := r.Param("dbname")

		err := dbe.DeleteDatabase(c, name)
		if err != nil {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases := admin_db.Group("/databases")

	// SCHEMAS

	databases.Handle("GET", "/:dbname/schemas", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)

		schemas, err := db.GetSchemas(c)
		if err == nil {
			JSON(w, http.StatusOK, schemas)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("POST", "/:dbname/schemas", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		var schemaInput database.Schema
		r.Bind(&schemaInput)

		schema, err := db.CreateSchema(c, schemaInput.Name)
		if err == nil {
			JSON(w, http.StatusCreated, schema)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("DELETE", "/:dbname/schemas/:schema", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		name := r.Param("schema")

		err := db.DeleteSchema(c, name, true)
		if err == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	// TABLES

	databases.Handle("GET", "/:dbname/tables", TableListHandler)

	databases.Handle("GET", "/:dbname/tables/:table", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		name := r.Param("table")

		table, err := db.GetTable(c, name)
		if err == nil {
			JSON(w, http.StatusOK, table)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("POST", "/:dbname/tables", TableCreateHandler)

	databases.Handle("PATCH", "/:dbname/tables/:table", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		var tableUpdate database.TableUpdate
		tableUpdate.Name = r.Param("table")
		r.Bind(&tableUpdate)

		table, err := db.UpdateTable(c, &tableUpdate)
		if err == nil {
			JSON(w, http.StatusOK, table)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("DELETE", "/:dbname/tables/:table", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		name := r.Param("table")

		err := db.DeleteTable(c, name)
		if err == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	// VIEWS

	databases.Handle("GET", "/:dbname/views", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)

		views, err := db.GetViews(c)
		if err == nil {
			JSON(w, http.StatusOK, views)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("GET", "/:dbname/views/:view", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		name := r.Param("view")

		view, err := db.GetView(c, name)
		if err == nil {
			JSON(w, http.StatusOK, view)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("POST", "/:dbname/views", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		var viewInput database.View
		r.Bind(&viewInput)

		view, err := db.CreateView(c, &viewInput)
		if err == nil {
			JSON(w, http.StatusCreated, view)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("DELETE", "/:dbname/views/:view", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		name := r.Param("view")

		err := db.DeleteView(c, name)
		if err != nil {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	// COLUMNS

	databases.Handle("GET", "/:dbname/tables/:table/columns", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		table := r.Param("table")

		columns, err := db.GetColumns(c, table)
		if err == nil {
			JSON(w, http.StatusOK, columns)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("POST", "/:dbname/tables/:table/columns", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		var columnInput database.Column
		columnInput.Table = r.Param("table")
		r.Bind(&columnInput)
		if columnInput.Type == "" {
			columnInput.Type = "text"
		}

		column, err := db.CreateColumn(c, &columnInput)
		if err == nil {
			JSON(w, http.StatusCreated, column)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("PATCH", "/:dbname/tables/:table/columns/:column", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		var columnUpdate database.ColumnUpdate
		columnUpdate.Table = r.Param("table")
		columnUpdate.Name = r.Param("column")
		r.Bind(&columnUpdate)

		column, err := db.UpdateColumn(c, &columnUpdate)
		if err == nil {
			JSON(w, http.StatusOK, column)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("DELETE", "/:dbname/tables/:table/columns/:column", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		table := r.Param("table")
		column := r.Param("column")

		err := db.DeleteColumn(c, table, column, false)
		if err != nil {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	// CONSTRAINTS

	databases.Handle("GET", "/:dbname/tables/:table/constraints", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		table := r.Param("table")

		constraints, err := db.GetConstraints(c, table)
		if err == nil {
			JSON(w, http.StatusOK, constraints)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("POST", "/:dbname/tables/:table/constraints", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		var constraintInput database.Constraint
		constraintInput.Table = r.Param("table")
		r.Bind(&constraintInput)

		constant, err := db.CreateConstraint(c, &constraintInput)
		if err == nil {
			JSON(w, http.StatusCreated, constant)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("DELETE", "/:dbname/tables/:table/constraints/:name", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		table := r.Param("table")
		name := r.Param("name")

		err := db.DeleteConstraint(c, table, name)
		if err != nil {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	// POLICIES

	databases.Handle("GET", "/:dbname/tables/:table/policies", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		table := r.Param("table")

		policies, err := db.GetPolicies(c, table)
		if err == nil {
			JSON(w, http.StatusOK, policies)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("POST", "/:dbname/tables/:table/policies", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		var policyInput database.Policy
		policyInput.Table = r.Param("table")
		r.Bind(&policyInput)

		policy, err := db.CreatePolicy(c, &policyInput)
		if err == nil {
			JSON(w, http.StatusCreated, policy)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("DELETE", "/:dbname/tables/:table/policies/:name", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		table := r.Param("table")
		name := r.Param("name")

		err := db.DeletePolicy(c, table, name)
		if err != nil {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	// FUNCTIONS

	databases.Handle("GET", "/:dbname/functions", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)

		policies, err := db.GetFunctions(c)
		if err == nil {
			JSON(w, http.StatusOK, policies)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("POST", "/:dbname/functions", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		var functionInput database.Function
		r.Bind(&functionInput)

		policy, err := db.CreateFunction(c, &functionInput)
		if err == nil {
			JSON(w, http.StatusCreated, policy)
		} else {
			WriteServerError(w, err)
			return err
		}
		return nil
	})

	databases.Handle("DELETE", "/:dbname/functions/:name", func(c context.Context, w heligo.ResponseWriter, r heligo.Request) error {
		db := database.GetDb(c)
		name := r.Param("name")

		err := db.DeleteFunction(c, name)
		if err != nil {
			WriteServerError(w, err)
			return err
		}
		return nil
	})
}
