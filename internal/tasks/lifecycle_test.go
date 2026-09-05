package tasks_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
	"github.com/GVALFER/WEBYCP/internal/tasks"
)

type taskAgent struct {
	account, user string
	entries       []tasks.Entry
	err           error
}

func (a *taskAgent) SyncTasks(_ context.Context, _, account, user string, entries []tasks.Entry) error {
	a.account, a.user, a.entries = account, user, entries
	return a.err
}

func TestLifecycleAndAccountIsolation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.InitAdmin(ctx, auth.NewUser{ID: "owner", Username: "admin", Email: "admin@example.test", Name: "Admin", PasswordHash: "hash", Role: "admin", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	node, err := store.EnsureLocal(ctx, "Test", "/tmp/test-agent.sock")
	if err != nil {
		t.Fatal(err)
	}
	accountService := accounts.NewService(store, store, nil, packages.NewService(store), func() {})
	account, _, err := accountService.Create(ctx, "Test account", node.ID, packages.DefaultID, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateStatus(ctx, account.ID, "active"); err != nil {
		t.Fatal(err)
	}
	agent := &taskAgent{}
	service := tasks.NewService(store, accountService, store, agent, func() {})
	task, job, err := service.Create(ctx, account.ID, "Hourly", "0 * * * *", "/usr/bin/true", services.Crontab, "owner", tasks.Command, false, true)
	if err != nil || job.Kind != jobs.KindTaskSync || strings.Contains(job.Payload, task.Command) {
		t.Fatalf("create task = %+v, job = %+v, error = %v", task, job, err)
	}
	if err := service.Sync(ctx, job); err != nil || agent.account != account.ID || agent.user != account.SystemUser || len(agent.entries) != 1 || agent.entries[0].Kind != tasks.Command {
		t.Fatalf("sync = %+v, error = %v", agent, err)
	}
	if _, _, err := service.Create(ctx, account.ID, "Denied", "0 * * * *", "/usr/bin/true", services.Crontab, "outsider", tasks.Command, false, true); !errors.Is(err, accounts.ErrForbidden) {
		t.Fatalf("cross-account create error = %v", err)
	}
	if _, _, err := service.Update(ctx, task.ID, account.ID, "Denied", task.Schedule, task.Command, services.Crontab, "outsider", tasks.Command, false, true); !errors.Is(err, accounts.ErrForbidden) {
		t.Fatalf("cross-account update error = %v", err)
	}
	if _, err := service.Delete(ctx, task.ID, "outsider", false); !errors.Is(err, accounts.ErrForbidden) {
		t.Fatalf("cross-account delete error = %v", err)
	}
	page, err := service.ScheduledTaskPage(ctx, "outsider", false, pagination.Query{Page: 1, Size: 10})
	if err != nil || page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("cross-account listing = %+v, error = %v", page, err)
	}
	for _, enabled := range []bool{false, true} {
		_, job, err = service.Update(ctx, task.ID, account.ID, "Updated", "30 * * * *", task.Command, services.Crontab, "owner", tasks.Command, false, enabled)
		if err != nil {
			t.Fatal(err)
		}
		if err := service.Sync(ctx, job); err != nil {
			t.Fatal(err)
		}
		current, err := store.ScheduledTask(ctx, task.ID)
		if err != nil || current.Name != "Updated" || current.Schedule != "30 * * * *" || current.Enabled != enabled {
			t.Fatalf("updated task = %+v, error = %v", current, err)
		}
		if enabled && (current.Status != "active" || len(agent.entries) != 1 || agent.entries[0].Schedule != current.Schedule) {
			t.Fatalf("enabled task = %+v, agent = %+v", current, agent)
		}
		if !enabled && (current.Status != "disabled" || len(agent.entries) != 0) {
			t.Fatalf("disabled task = %+v, agent = %+v", current, agent)
		}
	}
	agent.err = errors.New("agent unavailable")
	if err := service.Sync(ctx, job); err == nil {
		t.Fatal("Agent failure was ignored")
	}
	current, err := store.ScheduledTask(ctx, task.ID)
	if err != nil || current.Status != "error" {
		t.Fatalf("failed task = %+v, error = %v", current, err)
	}
	agent.err = nil
	if err := service.Sync(ctx, job); err != nil {
		t.Fatal(err)
	}
	job, err = service.Delete(ctx, task.ID, "owner", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Sync(ctx, job); err != nil || len(agent.entries) != 0 {
		t.Fatalf("delete sync = %+v, error = %v", agent, err)
	}
	if _, err := store.ScheduledTask(ctx, task.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted task error = %v", err)
	}
}
