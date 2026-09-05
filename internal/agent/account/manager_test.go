package account

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

type cleaner struct {
	deleted bool
}

type fileAccess struct {
	err   error
	calls []string
}

func (f *fileAccess) Disable(context.Context, string, string) error {
	f.calls = append(f.calls, "disable")
	return f.err
}

func (f *fileAccess) Enable(context.Context, string, string) error {
	f.calls = append(f.calls, "enable")
	return f.err
}

func (f *fileAccess) Delete(context.Context, string, string) error {
	f.calls = append(f.calls, "delete")
	return f.err
}

func (c *cleaner) Delete(context.Context, string) error {
	c.deleted = true
	return nil
}

func TestManagerDeletesRuntimeBeforeUser(t *testing.T) {
	runtime := &cleaner{}
	linux := NewLinux()
	linux.home = t.TempDir()
	linux.trash = filepath.Join(t.TempDir(), "trash")
	linux.lookup = func(name string) (*user.User, error) {
		if !runtime.deleted {
			t.Fatal("system user checked before runtime deletion")
		}
		return nil, user.UnknownUserError(name)
	}
	ftp := &fileAccess{}
	manager := New(linux, runtime, ftp)
	if err := manager.Delete(
		context.Background(),
		"0123456789abcdef0123456789abcdef",
		"wcp_0123456789ab",
	); err != nil {
		t.Fatal(err)
	}
	if len(ftp.calls) != 1 || ftp.calls[0] != "delete" {
		t.Fatal("FTP access was not removed")
	}
}

func TestManagerFTPFailure(t *testing.T) {
	const id = "0123456789abcdef0123456789abcdef"
	const name = "wcp_0123456789ab"
	linux := NewLinux()
	linux.home = t.TempDir()
	home := filepath.Join(linux.home, name)
	if err := os.Mkdir(home, 0o710); err != nil {
		t.Fatal(err)
	}
	linux.lookup = func(string) (*user.User, error) {
		return &user.User{Uid: "1000", Gid: "1000", Name: "WEBYCP-" + id, HomeDir: home}, nil
	}
	runtime := &cleaner{}
	ftp := &fileAccess{err: errors.New("FTP unavailable")}
	manager := New(linux, runtime, ftp)
	if err := manager.Disable(context.Background(), id, name); err == nil {
		t.Fatal("FTP revocation failure was hidden")
	}
	info, err := os.Stat(home)
	if err != nil || info.Mode().Perm() != 0 {
		t.Fatalf("home was not disabled: %v", err)
	}
	if err := manager.Delete(context.Background(), id, name); err == nil || runtime.deleted {
		t.Fatal("identity cleanup continued before FTP access was removed")
	}
}

func TestManagerEnableFailureRevokesAccess(t *testing.T) {
	const id = "0123456789abcdef0123456789abcdef"
	const name = "wcp_0123456789ab"
	uid, gid := testIDs()
	linux := NewLinux()
	linux.home = t.TempDir()
	home := filepath.Join(linux.home, name)
	if err := os.Mkdir(home, 0o710); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(home, 0o710)
	linux.lookup = func(string) (*user.User, error) {
		return &user.User{Uid: uid, Gid: gid, Name: "WEBYCP-" + id, HomeDir: home}, nil
	}
	linux.lookupGroup = func(string) (*user.Group, error) { return &user.Group{Gid: gid}, nil }
	ftp := &fileAccess{err: errors.New("FTP activation failed")}
	manager := New(linux, &cleaner{}, ftp)
	if err := manager.Enable(context.Background(), id, name); err == nil {
		t.Fatal("FTP activation failure was hidden")
	}
	if len(ftp.calls) != 2 || ftp.calls[0] != "enable" || ftp.calls[1] != "disable" {
		t.Fatal("failed enable did not revoke FTP access again")
	}
	info, err := os.Stat(home)
	if err != nil || info.Mode().Perm() != 0 {
		t.Fatal("failed enable left the Account home accessible")
	}
}
