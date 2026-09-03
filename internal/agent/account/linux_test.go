package account

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestEnsureCreatesMissingUser(t *testing.T) {
	accountID := "0123456789abcdef0123456789abcdef"
	uid, gid := testIDs()
	created := false
	var command string
	var args []string
	manager := NewLinux()
	manager.home = t.TempDir()
	manager.lookupGroup = func(name string) (*user.Group, error) {
		return &user.Group{Name: name, Gid: gid}, nil
	}
	manager.lookup = func(name string) (*user.User, error) {
		if !created {
			return nil, user.UnknownUserError(name)
		}
		return &user.User{
			Username: name, Uid: uid, Gid: gid, HomeDir: filepath.Join(manager.home, name),
			Name: "WEBYCP:" + accountID,
		}, nil
	}
	manager.run = func(_ context.Context, name string, values ...string) error {
		created = true
		command = name
		args = values
		return os.Mkdir(filepath.Join(manager.home, "wcp_0123456789ab"), 0o755)
	}

	err := manager.Ensure(context.Background(), accountID, "wcp_0123456789ab")
	if err != nil {
		t.Fatal(err)
	}
	if command != useraddPath {
		t.Fatalf("command = %q", command)
	}
	want := []string{
		"--create-home", "--home-dir", filepath.Join(manager.home, "wcp_0123456789ab"),
		"--shell", nologinPath, "--comment", "WEBYCP:" + accountID,
		"--user-group", "--", "wcp_0123456789ab",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
	for name, wantMode := range map[string]os.FileMode{
		"": 0o710, "web": 0o710, "logs": 0o750, "tmp": 0o700,
	} {
		info, err := os.Stat(filepath.Join(manager.home, "wcp_0123456789ab", name))
		if err != nil {
			t.Fatal(err)
		}
		if mode := info.Mode().Perm(); mode != wantMode {
			t.Fatalf("%s mode = %o, want %o", name, mode, wantMode)
		}
	}
}

func TestEnsureIsIdempotent(t *testing.T) {
	accountID := "0123456789abcdef0123456789abcdef"
	uid, gid := testIDs()
	manager := NewLinux()
	manager.home = t.TempDir()
	manager.lookupGroup = func(name string) (*user.Group, error) {
		return &user.Group{Name: name, Gid: gid}, nil
	}
	if err := os.Mkdir(filepath.Join(manager.home, "wcp_0123456789ab"), 0o750); err != nil {
		t.Fatal(err)
	}
	manager.lookup = func(name string) (*user.User, error) {
		return &user.User{
			Username: name, Uid: uid, Gid: gid, HomeDir: filepath.Join(manager.home, name),
			Name: "WEBYCP:" + accountID,
		}, nil
	}
	manager.run = func(context.Context, string, ...string) error {
		t.Fatal("useradd should not run for an existing user")
		return nil
	}

	if err := manager.Ensure(context.Background(), accountID, "wcp_0123456789ab"); err != nil {
		t.Fatal(err)
	}
}

func testIDs() (string, string) {
	uid := os.Getuid()
	gid := os.Getgid()
	if uid == 0 {
		uid = 1001
	}
	if gid == 0 {
		gid = 1001
	}
	return strconv.Itoa(uid), strconv.Itoa(gid)
}

func TestEnsureRejectsUnexpectedUser(t *testing.T) {
	accountID := "0123456789abcdef0123456789abcdef"
	manager := NewLinux()
	if err := manager.Ensure(context.Background(), accountID, "root"); err == nil {
		t.Fatal("expected invalid system user error")
	}
	manager.lookup = func(name string) (*user.User, error) {
		return &user.User{
			Username: name, Uid: "0", HomeDir: "/home/" + name,
			Name: "WEBYCP:" + accountID,
		}, nil
	}
	if err := manager.Ensure(context.Background(), accountID, "wcp_0123456789ab"); err == nil {
		t.Fatal("expected root user error")
	}
}

func TestEnsureRejectsUserOwnedByAnotherAccount(t *testing.T) {
	manager := NewLinux()
	manager.lookup = func(name string) (*user.User, error) {
		return &user.User{
			Username: name, Uid: "1001", HomeDir: "/home/" + name,
			Name: "another owner",
		}, nil
	}

	err := manager.Ensure(
		context.Background(),
		"0123456789abcdef0123456789abcdef",
		"wcp_0123456789ab",
	)
	if err == nil {
		t.Fatal("expected account ownership error")
	}
}

func TestDisableAndEnableAccountHome(t *testing.T) {
	accountID := "0123456789abcdef0123456789abcdef"
	systemUser := "wcp_0123456789ab"
	uid, gid := testIDs()
	manager := NewLinux()
	manager.home = t.TempDir()
	home := filepath.Join(manager.home, systemUser)
	if err := os.Mkdir(home, 0o710); err != nil {
		t.Fatal(err)
	}
	manager.lookup = func(string) (*user.User, error) {
		return &user.User{Username: systemUser, Uid: uid, Gid: gid, HomeDir: home, Name: "WEBYCP:" + accountID}, nil
	}
	manager.lookupGroup = func(string) (*user.Group, error) {
		return &user.Group{Gid: gid}, nil
	}
	if err := manager.Disable(context.Background(), accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(home)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0 {
		t.Fatalf("disabled home mode = %v", info.Mode().Perm())
	}
	if err := manager.Enable(context.Background(), accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(home)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o710 {
		t.Fatalf("enabled home mode = %v", info.Mode().Perm())
	}
}

func TestDeleteQuarantinesHomeAndIsRetrySafe(t *testing.T) {
	accountID := "0123456789abcdef0123456789abcdef"
	systemUser := "wcp_0123456789ab"
	uid, gid := testIDs()
	manager := NewLinux()
	manager.home = t.TempDir()
	manager.trash = filepath.Join(t.TempDir(), "trash")
	home := filepath.Join(manager.home, systemUser)
	if err := os.Mkdir(home, 0o710); err != nil {
		t.Fatal(err)
	}
	exists := true
	manager.lookup = func(string) (*user.User, error) {
		if !exists {
			return nil, user.UnknownUserError(systemUser)
		}
		return &user.User{Username: systemUser, Uid: uid, Gid: gid, HomeDir: home, Name: "WEBYCP:" + accountID}, nil
	}
	manager.run = func(_ context.Context, name string, args ...string) error {
		if name != userdelPath || !reflect.DeepEqual(args, []string{"--", systemUser}) {
			t.Fatalf("unexpected command: %s %#v", name, args)
		}
		exists = false
		return nil
	}
	for range 2 {
		if err := manager.Delete(context.Background(), accountID, systemUser); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(manager.trash, accountID)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(home); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("home still exists: %v", err)
	}
}
