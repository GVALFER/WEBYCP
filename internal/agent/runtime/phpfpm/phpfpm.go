package phpfpm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GVALFER/WEBYCP/internal/agent/configfile"
	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	agentruntime "github.com/GVALFER/WEBYCP/internal/agent/runtime"
	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

const (
	Version        = "8.3"
	defaultPools   = "/etc/php/8.3/fpm/pool.d"
	defaultRun     = "/run/php"
	phpFPMPath     = "/usr/sbin/php-fpm8.3"
	systemctlPath  = "/usr/bin/systemctl"
	phpFPMService  = "php8.3-fpm"
	configFileMode = 0o644
)

type Driver struct {
	pools  string
	runDir string
	run    func(context.Context, string, ...string) error
}

func New() *Driver {
	return &Driver{pools: defaultPools, runDir: defaultRun, run: execx.Run}
}

func (d *Driver) Ensure(ctx context.Context, account agentruntime.Account) (agentruntime.Pool, error) {
	if err := hostuser.ValidateNames(account.ID, account.SystemUser); err != nil {
		return agentruntime.Pool{}, err
	}
	if account.Version != Version {
		return agentruntime.Pool{}, &validate.Error{
			Field: "phpVersion", Message: "Unsupported PHP version",
		}
	}
	if err := validHome(account.Home, account.SystemUser); err != nil {
		return agentruntime.Pool{}, err
	}
	if err := configfile.EnsureDir(d.pools, 0o755); err != nil {
		return agentruntime.Pool{}, err
	}
	if err := configfile.EnsureDir(d.runDir, 0o755); err != nil {
		return agentruntime.Pool{}, err
	}

	socket := filepath.Join(d.runDir, "webycp-"+Version+"-"+account.ID+".sock")
	path := filepath.Join(d.pools, "webycp-"+account.ID+".conf")
	previous, err := configfile.Take(path)
	if err != nil {
		return agentruntime.Pool{}, err
	}
	if err := configfile.Write(path, render(account, socket), configFileMode); err != nil {
		return agentruntime.Pool{}, err
	}
	if err := d.run(ctx, phpFPMPath, "-t"); err != nil {
		return agentruntime.Pool{}, errors.Join(
			fmt.Errorf("validate PHP-FPM configuration: %w", err), previous.Restore(),
		)
	}
	if err := d.run(ctx, systemctlPath, "reload", phpFPMService); err != nil {
		rollbackErr := previous.Restore()
		validateErr := d.run(ctx, phpFPMPath, "-t")
		reloadErr := d.run(ctx, systemctlPath, "reload", phpFPMService)
		return agentruntime.Pool{}, errors.Join(
			fmt.Errorf("reload PHP-FPM: %w", err),
			rollbackErr,
			wrapRecoveryError("validate restored PHP-FPM configuration", validateErr),
			wrapRecoveryError("reload restored PHP-FPM configuration", reloadErr),
		)
	}
	return agentruntime.Pool{Socket: socket}, nil
}

func (d *Driver) Delete(ctx context.Context, accountID string) error {
	if err := validate.ID("accountId", accountID); err != nil {
		return err
	}
	path := filepath.Join(d.pools, "webycp-"+accountID+".conf")
	previous, err := configfile.Take(path)
	if err != nil {
		return err
	}
	if !previous.Exists {
		return nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove PHP-FPM pool: %w", err)
	}
	if err := d.run(ctx, phpFPMPath, "-t"); err != nil {
		return errors.Join(
			fmt.Errorf("validate PHP-FPM configuration: %w", err),
			previous.Restore(),
		)
	}
	if err := d.run(ctx, systemctlPath, "reload", phpFPMService); err != nil {
		rollbackErr := previous.Restore()
		validateErr := d.run(ctx, phpFPMPath, "-t")
		reloadErr := d.run(ctx, systemctlPath, "reload", phpFPMService)
		return errors.Join(
			fmt.Errorf("reload PHP-FPM: %w", err),
			rollbackErr,
			wrapRecoveryError("validate restored PHP-FPM configuration", validateErr),
			wrapRecoveryError("reload restored PHP-FPM configuration", reloadErr),
		)
	}
	return nil
}

func validHome(path, systemUser string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path ||
		filepath.Base(path) != systemUser || strings.ContainsAny(path, " \t\r\n;{}[]") {
		return &validate.Error{Field: "home", Message: "Account home is invalid"}
	}
	return nil
}

func render(account agentruntime.Account, socket string) []byte {
	return []byte(fmt.Sprintf(`[%s]
user = %s
group = %s

listen = %s
listen.owner = %s
listen.group = %s
listen.mode = 0660

pm = ondemand
pm.max_children = 5
pm.process_idle_timeout = 10s
pm.max_requests = 500

chdir = %s
clear_env = yes
catch_workers_output = yes
security.limit_extensions = .php

php_admin_flag[log_errors] = on
php_admin_value[error_log] = %s/logs/php-error.log
php_admin_value[open_basedir] = %s:/usr/share/php
php_admin_value[upload_tmp_dir] = %s/tmp
php_admin_value[session.save_path] = %s/tmp
`, "webycp-"+Version+"-"+account.ID, account.SystemUser, account.SystemUser,
		socket, hostuser.WebGroup, hostuser.WebGroup, account.Home,
		account.Home, account.Home, account.Home, account.Home,
	))
}

func wrapRecoveryError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
