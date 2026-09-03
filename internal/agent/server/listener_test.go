package server

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestListenRejectsRegularFile(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "agent.sock")
	if err := os.WriteFile(socket, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Listen(socket); err == nil {
		t.Fatal("Listen() error = nil, want regular file rejection")
	}

	content, err := os.ReadFile(socket)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep" {
		t.Fatalf("socket path content = %q, want keep", content)
	}
}

func TestListenRejectsActiveSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "agent.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	if _, _, err := Listen(socket); err == nil {
		t.Fatal("Listen() error = nil, want active socket rejection")
	}
}
