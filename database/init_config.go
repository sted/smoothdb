package database

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5"
)

// checkOutput receives the warnings and errors of the startup checks.
var checkOutput io.Writer = os.Stdout

// PrepareDatabase prepares the database for SmoothDb
func PrepareDatabase(adminURL string, dbConfig *Config) error {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())
	ctx = ContextWithDbConn(ctx, nil, conn)

	// Auth role
	authConfig, err := ParsePostgresURL(dbConfig.URL)
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "CREATE ROLE "+quote(authConfig.User)+" LOGIN NOINHERIT PASSWORD "+quoteLit(authConfig.Password))
	if err != nil {
		if IsExist(err) {
			checkAuthRole(ctx, authConfig.User)
		} else {
			return err
		}
	}
	// Anon role
	if dbConfig.AnonRole != "" {
		_, err := conn.Exec(ctx, "CREATE ROLE "+quote(dbConfig.AnonRole)+" NOLOGIN")
		if err != nil {
			if IsExist(err) {
				checkAnonRole(ctx, dbConfig.AnonRole)
			} else {
				return err
			}
		}
		// Grant anon to auth
		_, err = conn.Exec(ctx, "GRANT "+quote(dbConfig.AnonRole)+" TO "+quote(authConfig.User))
		if err != nil {
			return err
		}
	}
	_, err = conn.Exec(ctx, "CREATE DATABASE "+quote(SMOOTHDB))
	if err != nil && !IsExist(err) {
		return err
	}
	return nil
}

func checkAuthRole(ctx context.Context, name string) (canContinue bool) {
	auth, err := GetRole(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			fmt.Fprintf(checkOutput, "Error: the authenticator role %q does not exist.\n", name)
		} else {
			fmt.Fprintf(checkOutput, "Error: cannot check the authenticator role %q: %v\n", name, err)
		}
		return false
	}
	canContinue = true
	if auth.IsSuperUser || auth.CanCreateDatabases || auth.CanCreateRoles || auth.CanBypassRLS {
		fmt.Fprintln(checkOutput, "Warning: the authenticator role should have minimal privileges. Drop or alter the role if in production.")
	}
	if !auth.CanLogin {
		fmt.Fprintln(checkOutput, "Error: the authenticator role must be allowed to login directly. Drop or alter the role if in production.")
		canContinue = false
	}
	return canContinue
}

func checkAnonRole(ctx context.Context, name string) bool {
	anon, err := GetRole(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// anonymous requests fail on SET ROLE until it is created
			fmt.Fprintf(checkOutput, "Warning: the anonymous role %q does not exist.\n", name)
			return true
		}
		fmt.Fprintf(checkOutput, "Error: cannot check the anonymous role %q: %v\n", name, err)
		return false
	}
	if anon.IsSuperUser || anon.CanCreateDatabases || anon.CanCreateRoles || anon.CanBypassRLS {
		fmt.Fprintln(checkOutput, "Warning: the anonymous role should have minimal privileges. Drop or alter the role if in production.")
	}
	if anon.CanLogin {
		fmt.Fprintln(checkOutput, "Warning: the anonymous role shouldn't be allowed to login directly. Alter role with NOLOGIN if in production.")
	}
	return true
}

func CheckDatabase(dbConfig *Config) (bool, error) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbConfig.URL)
	if err != nil {
		return false, err
	}
	defer conn.Close(context.Background())
	ctx = ContextWithDbConn(ctx, nil, conn)
	authConfig, err := ParsePostgresURL(dbConfig.URL)
	if err != nil {
		return false, err
	}
	if !checkAuthRole(ctx, authConfig.User) {
		return false, nil
	}
	if dbConfig.AnonRole != "" && !checkAnonRole(ctx, dbConfig.AnonRole) {
		return false, nil
	}
	return true, nil
}
