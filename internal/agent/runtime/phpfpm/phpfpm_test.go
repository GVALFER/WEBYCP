package phpfpm

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentruntime "github.com/GVALFER/WEBYCP/internal/agent/runtime"
)

const testAccountID = "0123456789abcdef0123456789abcdef"

func TestEnsureInstallsValidatedPool(t *testing.T) {
	driver := testDriver(t)
	var commands []string
	driver.run = func(_ context.Context, name string, args ...string) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	account := testAccount(t)
	pool, err := driver.Ensure(context.Background(), account)
	if err != nil {
		t.Fatal(err)
	}
	wantSocket := filepath.Join(driver.runDir, "webycp-"+Version+"-"+testAccountID+".sock")
	if pool.Socket != wantSocket {
		t.Fatalf("socket = %q, want %q", pool.Socket, wantSocket)
	}
	data, err := os.ReadFile(filepath.Join(driver.pools, "webycp-"+testAccountID+".conf"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(data)
	for _, value := range []string{
		"[webycp-" + Version + "-" + testAccountID + "]",
		"user = wcp_0123456789ab",
		"listen = " + wantSocket,
		"listen.group = www-data",
		"php_admin_value[open_basedir] = " + account.Home + ":/usr/share/php",
	} {
		if !strings.Contains(config, value) {
			t.Fatalf("config is missing %q:\n%s", value, config)
		}
	}
	want := []string{phpFPMPath + " -t", systemctlPath + " reload " + phpFPMService}
	if strings.Join(commands, "|") != strings.Join(want, "|") {
		t.Fatalf("commands = %#v, want %#v", commands, want)
	}
}

func TestEnsureRestoresConfigWhenValidationFails(t *testing.T) {
	driver := testDriver(t)
	path := filepath.Join(driver.pools, "webycp-"+testAccountID+".conf")
	if err := os.WriteFile(path, []byte("known-good\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	driver.run = func(_ context.Context, name string, _ ...string) error {
		if name == phpFPMPath {
			return errors.New("invalid config")
		}
		return nil
	}
	if _, err := driver.Ensure(context.Background(), testAccount(t)); err == nil {
		t.Fatal("expected validation error")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "known-good\n" {
		t.Fatalf("restored config = %q, error = %v", data, err)
	}
}

func TestEnsureRemovesNewPoolWhenReloadFails(t *testing.T) {
	driver := testDriver(t)
	reloads := 0
	driver.run = func(_ context.Context, name string, _ ...string) error {
		if name == systemctlPath {
			reloads++
			if reloads == 1 {
				return errors.New("reload failed")
			}
		}
		return nil
	}
	if _, err := driver.Ensure(context.Background(), testAccount(t)); err == nil {
		t.Fatal("expected reload error")
	}
	path := filepath.Join(driver.pools, "webycp-"+testAccountID+".conf")
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rollback left pool config: %v", err)
	}
	if reloads != 2 {
		t.Fatalf("reloads = %d, want recovery reload", reloads)
	}
}

func TestEnsureIsIdempotent(t *testing.T) {
	driver := testDriver(t)
	driver.run = func(context.Context, string, ...string) error { return nil }
	account := testAccount(t)
	for range 2 {
		pool, err := driver.Ensure(context.Background(), account)
		if err != nil {
			t.Fatal(err)
		}
		if pool.Socket != filepath.Join(driver.runDir, "webycp-"+Version+"-"+testAccountID+".sock") {
			t.Fatalf("unexpected socket: %s", pool.Socket)
		}
	}
}

func TestEnsureRejectsUnsupportedVersion(t *testing.T) {
	driver := testDriver(t)
	account := testAccount(t)
	account.Version = "8.4"
	if _, err := driver.Ensure(context.Background(), account); err == nil {
		t.Fatal("expected unsupported PHP version error")
	}
}

func TestDeleteRemovesValidatedPool(t *testing.T) {
	driver := testDriver(t)
	path := filepath.Join(driver.pools, "webycp-"+testAccountID+".conf")
	if err := os.WriteFile(path, []byte("pool\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var commands []string
	driver.run = func(_ context.Context, name string, args ...string) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	if err := driver.Delete(context.Background(), testAccountID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pool still exists: %v", err)
	}
	want := []string{phpFPMPath + " -t", systemctlPath + " reload " + phpFPMService}
	if strings.Join(commands, "|") != strings.Join(want, "|") {
		t.Fatalf("commands = %#v, want %#v", commands, want)
	}
}

func TestDeleteRestoresPoolWhenValidationFails(t *testing.T) {
	driver := testDriver(t)
	path := filepath.Join(driver.pools, "webycp-"+testAccountID+".conf")
	if err := os.WriteFile(path, []byte("known-good\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	driver.run = func(context.Context, string, ...string) error {
		return errors.New("invalid config")
	}
	if err := driver.Delete(context.Background(), testAccountID); err == nil {
		t.Fatal("expected validation error")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "known-good\n" {
		t.Fatalf("restored config = %q, error = %v", data, err)
	}
}

func testDriver(t *testing.T) *Driver {
	t.Helper()
	root := t.TempDir()
	driver := New()
	driver.pools = filepath.Join(root, "pool.d")
	driver.runDir = filepath.Join(root, "run")
	if err := os.MkdirAll(driver.pools, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(driver.runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return driver
}

func testAccount(t *testing.T) agentruntime.Account {
	t.Helper()
	home := filepath.Join(t.TempDir(), "wcp_0123456789ab")
	return agentruntime.Account{
		ID: testAccountID, SystemUser: "wcp_0123456789ab", Home: home, Version: Version,
	}
}
