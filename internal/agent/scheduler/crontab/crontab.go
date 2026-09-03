package crontab

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GVALFER/WEBYCP/internal/agent/configfile"
	"github.com/GVALFER/WEBYCP/internal/agent/scheduler"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

const defaultDir = "/etc/cron.d"

type Driver struct {
	dir string
}

func New() *Driver {
	return &Driver{dir: defaultDir}
}

func (d *Driver) Sync(_ context.Context, accountID, systemUser string, entries []scheduler.Entry) error {
	if validate.ID("accountId", accountID) != nil || validate.SystemUser(systemUser) != nil {
		return &validate.Error{Field: "accountId", Message: "The account identity is invalid"}
	}
	if len(entries) > 100 {
		return &validate.Error{Field: "entries", Message: "Too many cron entries"}
	}
	if err := configfile.EnsureDir(d.dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(d.dir, "webycp-"+accountID)
	if len(entries) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove account crontab: %w", err)
		}
		return nil
	}
	var contents strings.Builder
	contents.WriteString("# Managed by WEBYCP. Manual changes will be replaced.\n")
	contents.WriteString("SHELL=/bin/sh\n")
	contents.WriteString("PATH=/usr/local/bin:/usr/bin:/bin\n")
	contents.WriteString("MAILTO=\"\"\n")
	contents.WriteString("HOME=/home/" + systemUser + "\n\n")
	for _, entry := range entries {
		if validate.ID("cronJobId", entry.ID) != nil {
			return &validate.Error{Field: "id", Message: "Cron entry ID is invalid"}
		}
		schedule, err := validate.CronSchedule(entry.Schedule, false)
		if err != nil || schedule != entry.Schedule {
			return &validate.Error{Field: "schedule", Message: "Cron schedule is invalid"}
		}
		command, err := validate.CronCommand(entry.Command)
		if err != nil || command != entry.Command {
			return &validate.Error{Field: "command", Message: "Cron command is invalid"}
		}
		contents.WriteString("# " + entry.ID + "\n")
		contents.WriteString(schedule + " " + systemUser + " cd -- /home/" + systemUser + " && " + command + "\n")
	}
	if err := configfile.Write(path, []byte(contents.String()), 0o644); err != nil {
		return fmt.Errorf("install account crontab: %w", err)
	}
	return nil
}
