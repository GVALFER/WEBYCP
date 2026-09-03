package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

func Listen(socket string) (net.Listener, func(), error) {
	if err := prepareSocket(socket); err != nil {
		return nil, nil, err
	}

	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, nil, fmt.Errorf("listen on agent socket: %w", err)
	}

	if err := os.Chmod(socket, 0o660); err != nil {
		_ = listener.Close()
		return nil, nil, fmt.Errorf("set agent socket permissions: %w", err)
	}

	cleanup := func() {
		_ = listener.Close()
		_ = os.Remove(socket)
	}

	return listener, cleanup, nil
}

func prepareSocket(socket string) error {
	if err := os.MkdirAll(filepath.Dir(socket), 0o750); err != nil {
		return fmt.Errorf("create agent socket directory: %w", err)
	}

	info, err := os.Lstat(socket)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect agent socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("agent socket path exists and is not a socket: %s", socket)
	}
	connection, dialErr := net.DialTimeout("unix", socket, 250*time.Millisecond)
	if dialErr == nil {
		_ = connection.Close()
		return fmt.Errorf("agent socket is already active: %s", socket)
	}

	if err := os.Remove(socket); err != nil {
		return fmt.Errorf("remove stale agent socket: %w", err)
	}

	return nil
}
