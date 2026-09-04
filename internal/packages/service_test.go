package packages_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

func TestPackageCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := packages.NewService(store)

	value := packages.Package{Name: "Starter", Limits: packages.Limits{
		Websites: 2, Domains: 2, Aliases: 4, Databases: 2, DatabaseUsers: 2,
		ScheduledTasks: 4, BackupPlans: 1, BackupRetention: 3,
	}}
	created, err := service.Create(ctx, value)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Name != value.Name || created.Limits != value.Limits {
		t.Fatalf("created Package = %+v", created)
	}
	if _, err := service.Create(ctx, value); !errors.Is(err, packages.ErrNameExists) {
		t.Fatalf("duplicate error = %v", err)
	}

	value.Name = "Starter Plus"
	value.Limits.Websites = 1
	value.Limits.Domains = 1
	updated, err := service.Update(ctx, created.ID, value)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != value.Name || updated.Limits.Websites != 1 {
		t.Fatalf("updated Package = %+v", updated)
	}

	page, err := service.Page(ctx, pagination.Query{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("Package total = %d, want default plus created", page.Total)
	}
	if err := service.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Package(ctx, created.ID); err == nil {
		t.Fatal("deleted Package still exists")
	}
}

func TestPackageRequiresEnoughPrimaryDomains(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := packages.NewService(store)

	_, err = service.Create(ctx, packages.Package{Name: "Invalid", Limits: packages.Limits{
		Websites: 2, Domains: 1, BackupRetention: 1,
	}})
	if err == nil {
		t.Fatal("created a Package with fewer domains than Websites")
	}
}
