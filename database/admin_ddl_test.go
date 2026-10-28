package database

import (
	"context"
	"testing"
)

// The /admin DDL helpers had no test: these call each one directly. Every
// object they create carries the lane6_ prefix and is dropped at the end.

// GetUsers ran usersQuery, which filters on the authenticator role as $1,
// without passing it, so every GET /admin/users failed.
func TestAdminGetUsers(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	DeleteUser(ctx, "lane6_user")
	if _, err := CreateUser(ctx, &User{Name: "lane6_user"}); err != nil {
		t.Fatal(err)
	}
	defer DeleteUser(ctx, "lane6_user")

	users, err := GetUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, u := range users {
		if u.Name == "lane6_user" {
			found = true
		}
	}
	if !found {
		t.Errorf("lane6_user missing from GetUsers: %+v", users)
	}
}
