package fsx

import (
	"errors"
	"fmt"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Dir struct {
	fd int
}

func OpenDir(path string) (*Dir, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open directory %s: %w", path, err)
	}
	return &Dir{fd: fd}, nil
}

func (d *Dir) Close() error {
	return unix.Close(d.fd)
}

func (d *Dir) Open(name string) (*Dir, error) {
	if err := validName(name); err != nil {
		return nil, err
	}
	fd, err := unix.Openat(
		d.fd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("open child directory %s: %w", name, err)
	}
	return &Dir{fd: fd}, nil
}

func (d *Dir) Configure(mode uint32, uid, gid int) error {
	if err := unix.Fchown(d.fd, uid, gid); err != nil {
		return fmt.Errorf("set directory owner: %w", err)
	}
	if err := unix.Fchmod(d.fd, mode); err != nil {
		return fmt.Errorf("set directory mode: %w", err)
	}
	return nil
}

func (d *Dir) Ensure(name string, mode uint32, uid, gid int) error {
	if err := validName(name); err != nil {
		return err
	}
	if err := unix.Mkdirat(d.fd, name, mode); err != nil && !errors.Is(err, unix.EEXIST) {
		return fmt.Errorf("create child directory %s: %w", name, err)
	}
	child, err := d.Open(name)
	if err != nil {
		return err
	}
	defer child.Close()
	if err := child.Configure(mode, uid, gid); err != nil {
		return fmt.Errorf("configure child directory %s: %w", name, err)
	}
	return nil
}

func (d *Dir) Rename(name string, target *Dir, targetName string) error {
	if err := validName(name); err != nil {
		return err
	}
	if err := validName(targetName); err != nil {
		return err
	}
	if existing, err := target.Open(targetName); err == nil {
		existing.Close()
		return fmt.Errorf("target directory %s already exists", targetName)
	} else if !errors.Is(err, unix.ENOENT) {
		return err
	}
	if err := unix.Renameat(d.fd, name, target.fd, targetName); err != nil {
		return fmt.Errorf("move child directory %s: %w", name, err)
	}
	return nil
}

func validName(name string) error {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return fmt.Errorf("invalid child directory name %q", name)
	}
	return nil
}
