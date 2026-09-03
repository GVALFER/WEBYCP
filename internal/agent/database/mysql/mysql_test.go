package mysql

import (
	"context"
	"strings"
	"testing"
)

func TestLifecycleUsesValidatedTypedStatements(t *testing.T) {
	driver := New()
	var statements []string
	driver.run = func(_ context.Context, name string, args ...string) error {
		if name != mysqlPath || len(args) != 5 || args[3] != "--execute" {
			t.Fatalf("unexpected MySQL invocation: %s %#v", name, args)
		}
		statements = append(statements, args[4])
		return nil
	}
	database := "wcp_01234567_app"
	user := "wcp_01234567_admin"
	password := strings.Repeat("A", 32)
	operations := []func() error{
		func() error { return driver.EnsureDatabase(context.Background(), database) },
		func() error { return driver.EnsureUser(context.Background(), user, password) },
		func() error { return driver.EnsureGrant(context.Background(), database, user) },
		func() error { return driver.DeleteGrant(context.Background(), database, user) },
		func() error { return driver.DeleteUser(context.Background(), user) },
		func() error { return driver.DeleteDatabase(context.Background(), database) },
	}
	for _, operation := range operations {
		if err := operation(); err != nil {
			t.Fatal(err)
		}
	}
	if len(statements) != 7 {
		t.Fatalf("statements = %d, want 7", len(statements))
	}
	if !strings.Contains(statements[0], "CREATE DATABASE IF NOT EXISTS `"+database+"`") ||
		!strings.Contains(statements[3], "GRANT ALL PRIVILEGES") ||
		!strings.Contains(statements[4], "REVOKE IF EXISTS") ||
		!strings.Contains(statements[4], "IGNORE UNKNOWN USER") {
		t.Fatalf("unexpected statements: %#v", statements)
	}
}

func TestRejectsUnsafeNamesAndPasswords(t *testing.T) {
	driver := New()
	driver.run = func(context.Context, string, ...string) error {
		t.Fatal("invalid input must not reach MySQL")
		return nil
	}
	if err := driver.EnsureDatabase(context.Background(), "app`; DROP DATABASE mysql"); err == nil {
		t.Fatal("expected invalid database name")
	}
	if err := driver.EnsureUser(context.Background(), "wcp_01234567_admin", "short"); err == nil {
		t.Fatal("expected invalid password")
	}
}
