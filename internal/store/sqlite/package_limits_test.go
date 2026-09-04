package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/backups"
	cronjob "github.com/GVALFER/WEBYCP/internal/cron"
	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/websites"
)

func TestPackageCountLimits(t *testing.T) {
	tests := []struct {
		name     string
		resource packages.Resource
		prepare  func(*testing.T, context.Context, *Store, accounts.Account)
		create   func(context.Context, *Store, accounts.Account, int) error
	}{
		{
			name: "websites", resource: packages.Websites,
			create: func(ctx context.Context, store *Store, account accounts.Account, n int) error {
				_, _, _, err := store.CreateWebsiteProvision(ctx,
					testWebsite(account, n), testDomain(n, "primary"), limitJob(n, account.NodeID, jobs.KindWebsiteCreate),
				)
				return err
			},
		},
		{
			name: "domains", resource: packages.Domains,
			prepare: func(t *testing.T, ctx context.Context, store *Store, account accounts.Account) {
				setLimit(t, ctx, store, account, func(value *packages.Limits) { value.Websites = 2 })
			},
			create: func(ctx context.Context, store *Store, account accounts.Account, n int) error {
				_, _, _, err := store.CreateWebsiteProvision(ctx,
					testWebsite(account, n), testDomain(n, "primary"), limitJob(n, account.NodeID, jobs.KindWebsiteCreate),
				)
				return err
			},
		},
		{
			name: "aliases", resource: packages.Aliases,
			prepare: func(t *testing.T, ctx context.Context, store *Store, account accounts.Account) {
				website := testWebsite(account, 10)
				if _, _, _, err := store.CreateWebsiteProvision(ctx, website, testDomain(10, "primary"), limitJob(10, account.NodeID, jobs.KindWebsiteCreate)); err != nil {
					t.Fatal(err)
				}
			},
			create: func(ctx context.Context, store *Store, account accounts.Account, n int) error {
				domain := testDomain(n, "alias")
				domain.WebsiteID = limitID(10)
				_, _, err := store.CreateWebsiteDomainProvision(ctx, domain, limitJob(n, account.NodeID, jobs.KindWebsiteDomainCreate))
				return err
			},
		},
		{
			name: "databases", resource: packages.Databases,
			create: func(ctx context.Context, store *Store, account accounts.Account, n int) error {
				_, _, err := store.CreateDatabase(ctx, testDatabase(account, n), limitJob(n, account.NodeID, jobs.KindDatabaseCreate))
				return err
			},
		},
		{
			name: "database users", resource: packages.DatabaseUsers,
			create: func(ctx context.Context, store *Store, account accounts.Account, n int) error {
				_, _, err := store.CreateUser(ctx, databases.User{
					ID: limitID(n), AccountID: account.ID, NodeID: account.NodeID,
					Name: fmt.Sprintf("user_%d", n), SystemName: fmt.Sprintf("wcp_user_%d", n),
					Driver: services.MySQL, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
				}, limitJob(n, account.NodeID, jobs.KindDatabaseUserCreate))
				return err
			},
		},
		{
			name: "scheduled tasks", resource: packages.ScheduledTasks,
			create: func(ctx context.Context, store *Store, account accounts.Account, n int) error {
				_, _, err := store.CreateCronJob(ctx, testCron(account, n), limitJob(n, account.NodeID, jobs.KindCronSync))
				return err
			},
		},
		{
			name: "backup plans", resource: packages.BackupPlans,
			create: func(ctx context.Context, store *Store, account accounts.Account, n int) error {
				_, err := store.CreateBackupPlan(ctx, testPlan(account, n, 1))
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, store, account := limitStore(t)
			if test.prepare != nil {
				test.prepare(t, ctx, store, account)
			}
			if err := test.create(ctx, store, account, 1); err != nil {
				t.Fatalf("create at limit: %v", err)
			}
			err := test.create(ctx, store, account, 2)
			var limit *packages.LimitError
			if !errors.As(err, &limit) || limit.Resource != test.resource || limit.Limit != 1 {
				t.Fatalf("limit error = %#v, want %s limit 1", err, test.resource)
			}
		})
	}
}

func TestConcurrentCreatesCannotExceedPackageLimit(t *testing.T) {
	ctx, store, account := limitStore(t)
	errorsByCreate := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	for n := 1; n <= 2; n++ {
		go func(index int) {
			ready.Done()
			<-start
			_, _, err := store.CreateDatabase(ctx, testDatabase(account, index), limitJob(index, account.NodeID, jobs.KindDatabaseCreate))
			errorsByCreate <- err
		}(n)
	}
	ready.Wait()
	close(start)

	succeeded, limited := 0, 0
	for range 2 {
		err := <-errorsByCreate
		var limit *packages.LimitError
		switch {
		case err == nil:
			succeeded++
		case errors.As(err, &limit):
			limited++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if succeeded != 1 || limited != 1 {
		t.Fatalf("succeeded=%d limited=%d, want 1 and 1", succeeded, limited)
	}
	overview, err := store.AccountOverview(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Usage.Databases != 1 {
		t.Fatalf("database usage = %d, want 1", overview.Usage.Databases)
	}
}

func TestLowerPackageKeepsExistingResourcesUsable(t *testing.T) {
	ctx, store, account := limitStore(t)
	cron := testCron(account, 1)
	if _, _, err := store.CreateCronJob(ctx, cron, limitJob(1, account.NodeID, jobs.KindCronSync)); err != nil {
		t.Fatal(err)
	}
	setLimit(t, ctx, store, account, func(value *packages.Limits) { value.ScheduledTasks = 0 })

	cron.Name = "Updated task"
	if _, _, err := store.UpdateCronJob(ctx, cron, limitJob(2, account.NodeID, jobs.KindCronSync)); err != nil {
		t.Fatalf("update existing over-limit resource: %v", err)
	}
	if _, _, err := store.CreateCronJob(ctx, testCron(account, 2), limitJob(3, account.NodeID, jobs.KindCronSync)); err == nil {
		t.Fatal("new resource was created above the lowered limit")
	}
	overview, err := store.AccountOverview(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Usage.ScheduledTasks != 1 || overview.Package.Limits.ScheduledTasks != 0 {
		t.Fatalf("overview = %+v", overview)
	}
}

func TestBackupRetentionLimit(t *testing.T) {
	ctx, store, account := limitStore(t)
	setLimit(t, ctx, store, account, func(value *packages.Limits) { value.BackupRetention = 2 })
	if _, err := store.CreateBackupPlan(ctx, testPlan(account, 1, 3)); err == nil {
		t.Fatal("created a plan above the retention limit")
	}
	plan := testPlan(account, 1, 2)
	if _, err := store.CreateBackupPlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	setLimit(t, ctx, store, account, func(value *packages.Limits) { value.BackupRetention = 1 })
	if _, err := store.UpdateBackupPlan(ctx, plan, 2); err != nil {
		t.Fatalf("kept existing retention above lowered limit: %v", err)
	}
	plan.RetentionCount = 3
	if _, err := store.UpdateBackupPlan(ctx, plan, 2); err == nil {
		t.Fatal("increased retention above the lowered limit")
	}
	plan.RetentionCount = 1
	if _, err := store.UpdateBackupPlan(ctx, plan, 2); err != nil {
		t.Fatalf("reduced retention to package limit: %v", err)
	}
}

func TestAssignedPackageCannotBeDeleted(t *testing.T) {
	ctx, store, account := limitStore(t)
	overview, err := store.AccountOverview(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeletePackage(ctx, overview.Package.ID); !errors.Is(err, packages.ErrInUse) {
		t.Fatalf("delete assigned Package error = %v", err)
	}
}

func TestRestoreCannotBypassPackageLimits(t *testing.T) {
	ctx, store, account := limitStore(t)
	metadata := backupfmt.Metadata{Version: backupfmt.Version, AccountID: account.ID}
	for n := 1; n <= 2; n++ {
		website := testWebsite(account, n)
		domain := testDomain(n, "primary")
		metadata.Websites = append(metadata.Websites, backupfmt.Website{
			ID: website.ID, NodeID: website.NodeID, Name: website.Name,
			Kind: website.Kind, DocumentRoot: website.DocumentRoot,
			WebDriver: website.WebDriver, RuntimeDriver: website.RuntimeDriver,
			RuntimeVersion: website.RuntimeVersion, Enabled: true,
			Domains: []backupfmt.WebsiteDomain{{
				ID: domain.ID, Hostname: domain.Hostname, Kind: domain.Kind, Enabled: true,
			}},
		})
	}
	err := store.RestoreMetadata(ctx, metadata)
	var limit *packages.LimitError
	if !errors.As(err, &limit) || limit.Resource != packages.Websites {
		t.Fatalf("restore limit error = %v", err)
	}
	overview, err := store.AccountOverview(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Usage.Websites != 0 {
		t.Fatalf("partial restore committed %d Websites", overview.Usage.Websites)
	}
}

func limitStore(t *testing.T) (context.Context, *Store, accounts.Account) {
	t.Helper()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	now := time.Now().UTC()
	if created, err := store.InitAdmin(ctx, auth.NewUser{
		ID: limitID(100), Username: "admin", Email: "admin@example.com", Name: "Admin",
		PasswordHash: "hash", Role: "admin", CreatedAt: now,
	}); err != nil || !created {
		t.Fatalf("create user: created=%v error=%v", created, err)
	}
	node, err := store.EnsureLocal(ctx, "test", "/tmp/webycp-limit-agent.sock")
	if err != nil {
		t.Fatal(err)
	}
	service := packages.NewService(store)
	pkg, err := service.Create(ctx, packages.Package{Name: "Limited", Limits: packages.Limits{
		Websites: 1, Domains: 1, Aliases: 1, Databases: 1, DatabaseUsers: 1,
		ScheduledTasks: 1, BackupPlans: 1, BackupRetention: 1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	account := accounts.Account{
		ID: limitID(101), NodeID: node.ID, Name: "Limited account", SystemUser: "wcp_limited",
		Status: "active", Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if _, _, err := store.CreateProvision(ctx, account, limitID(100), pkg.ID, limitJob(100, node.ID, jobs.KindAccountCreate)); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateStatus(ctx, account.ID, "active"); err != nil {
		t.Fatal(err)
	}
	return ctx, store, account
}

func setLimit(t *testing.T, ctx context.Context, store *Store, account accounts.Account, change func(*packages.Limits)) {
	t.Helper()
	overview, err := store.AccountOverview(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	change(&overview.Package.Limits)
	overview.Package.UpdatedAt = time.Now().UTC()
	if _, err := store.UpdatePackage(ctx, overview.Package); err != nil {
		t.Fatal(err)
	}
}

func testWebsite(account accounts.Account, n int) websites.Website {
	now := time.Now().UTC()
	return websites.Website{
		ID: limitID(n), AccountID: account.ID, NodeID: account.NodeID,
		Name: fmt.Sprintf("Website %d", n), Kind: "php",
		DocumentRoot: fmt.Sprintf("/home/%s/web/site%d.example.com/public_html", account.SystemUser, n),
		WebDriver:    "nginx", RuntimeDriver: "phpfpm", RuntimeVersion: "8.3",
		Status: "pending", Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
}

func testDomain(n int, kind string) websites.WebsiteDomain {
	now := time.Now().UTC()
	return websites.WebsiteDomain{
		ID: limitID(1000 + n), WebsiteID: limitID(n), Hostname: fmt.Sprintf("site%d.example.com", n),
		Kind: kind, Status: "pending", Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
}

func testDatabase(account accounts.Account, n int) databases.Database {
	now := time.Now().UTC()
	return databases.Database{
		ID: limitID(2000 + n), AccountID: account.ID, NodeID: account.NodeID,
		Name: fmt.Sprintf("db_%d", n), SystemName: fmt.Sprintf("wcp_db_%d", n),
		Driver: services.MySQL, Status: "pending", CreatedAt: now, UpdatedAt: now,
	}
}

func testCron(account accounts.Account, n int) cronjob.CronJob {
	now := time.Now().UTC()
	return cronjob.CronJob{
		ID: limitID(3000 + n), AccountID: account.ID, NodeID: account.NodeID,
		Name: fmt.Sprintf("Task %d", n), Schedule: "0 * * * *", Command: "/usr/bin/true",
		SchedulerDriver: services.Crontab, Enabled: true, Status: "pending",
		CreatedAt: now, UpdatedAt: now,
	}
}

func testPlan(account accounts.Account, n int, retention int64) backups.Plan {
	now := time.Now().UTC()
	return backups.Plan{
		ID: limitID(4000 + n), AccountID: account.ID, NodeID: account.NodeID,
		Name: fmt.Sprintf("Plan %d", n), RetentionCount: retention,
		StorageDriver: services.Local, IncludeFiles: true, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
}

func limitJob(n int, nodeID, kind string) jobs.Job {
	return jobs.Job{
		ID: limitID(5000 + n), NodeID: nodeID, UserID: limitID(100),
		Kind: kind, Payload: "{}", MaxAttempts: 1, CreatedAt: time.Now().UTC(),
	}
}

func limitID(n int) string {
	return fmt.Sprintf("%032x", n)
}
