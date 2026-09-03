package mysql

import (
	"context"
	"fmt"
	"regexp"

	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

const mysqlPath = "/usr/bin/mysql"

var passwordPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,128}$`)

type Driver struct {
	run func(context.Context, string, ...string) error
}

func New() *Driver {
	return &Driver{run: execx.Run}
}

func (d *Driver) EnsureDatabase(ctx context.Context, name string) error {
	if err := validate.DatabaseSystemName(name); err != nil {
		return err
	}
	return d.sql(ctx, "CREATE DATABASE IF NOT EXISTS `"+name+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
}

func (d *Driver) DeleteDatabase(ctx context.Context, name string) error {
	if err := validate.DatabaseSystemName(name); err != nil {
		return err
	}
	return d.sql(ctx, "DROP DATABASE IF EXISTS `"+name+"`")
}

func (d *Driver) EnsureUser(ctx context.Context, name, password string) error {
	if err := validate.DatabaseSystemName(name); err != nil {
		return err
	}
	if !passwordPattern.MatchString(password) {
		return &validate.Error{Field: "password", Message: "Generated database password is invalid"}
	}
	account := "'" + name + "'@'localhost'"
	if err := d.sql(ctx, "CREATE USER IF NOT EXISTS "+account+" IDENTIFIED BY '"+password+"'"); err != nil {
		return err
	}
	return d.sql(ctx, "ALTER USER "+account+" IDENTIFIED BY '"+password+"'")
}

func (d *Driver) DeleteUser(ctx context.Context, name string) error {
	if err := validate.DatabaseSystemName(name); err != nil {
		return err
	}
	return d.sql(ctx, "DROP USER IF EXISTS '"+name+"'@'localhost'")
}

func (d *Driver) EnsureGrant(ctx context.Context, databaseName, userName string) error {
	if err := validate.DatabaseSystemName(databaseName); err != nil {
		return err
	}
	if err := validate.DatabaseSystemName(userName); err != nil {
		return err
	}
	return d.sql(ctx, "GRANT ALL PRIVILEGES ON `"+databaseName+"`.* TO '"+userName+"'@'localhost'")
}

func (d *Driver) DeleteGrant(ctx context.Context, databaseName, userName string) error {
	if err := validate.DatabaseSystemName(databaseName); err != nil {
		return err
	}
	if err := validate.DatabaseSystemName(userName); err != nil {
		return err
	}
	return d.sql(ctx, "REVOKE IF EXISTS ALL PRIVILEGES ON `"+databaseName+"`.* FROM '"+userName+"'@'localhost' IGNORE UNKNOWN USER")
}

func (d *Driver) sql(ctx context.Context, statement string) error {
	if err := d.run(ctx, mysqlPath, "--protocol=socket", "--batch", "--skip-column-names", "--execute", statement); err != nil {
		return fmt.Errorf("execute MySQL operation: %w", err)
	}
	return nil
}
