package configfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Snapshot struct {
	path   string
	Data   []byte
	Mode   os.FileMode
	Exists bool
}

func EnsureDir(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect config directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("config path is not a directory: %s", path)
	}
	return nil
}

func Take(path string) (Snapshot, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Snapshot{path: path}, nil
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("inspect config file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Snapshot{}, fmt.Errorf("config path is not a regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read config file: %w", err)
	}
	return Snapshot{
		path: path, Data: data, Mode: info.Mode().Perm(), Exists: true,
	}, nil
}

func Write(path string, data []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".webycp-*")
	if err != nil {
		return fmt.Errorf("create config candidate: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return fmt.Errorf("set config candidate mode: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write config candidate: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync config candidate: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close config candidate: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("install config candidate: %w", err)
	}
	return nil
}

func (s Snapshot) Restore() error {
	if !s.Exists {
		if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove config candidate: %w", err)
		}
		return nil
	}
	if err := Write(s.path, s.Data, s.Mode); err != nil {
		return fmt.Errorf("restore config file: %w", err)
	}
	return nil
}
