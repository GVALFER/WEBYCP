package account

import (
	"context"
	"os/user"
	"path/filepath"
	"testing"
)

type cleaner struct {
	deleted bool
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
	manager := New(linux, runtime)
	if err := manager.Delete(
		context.Background(),
		"0123456789abcdef0123456789abcdef",
		"wcp_0123456789ab",
	); err != nil {
		t.Fatal(err)
	}
}
