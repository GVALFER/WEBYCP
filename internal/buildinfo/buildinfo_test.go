package buildinfo

import (
	"bytes"
	"testing"
)

func TestShow(t *testing.T) {
	previousVersion := Version
	previousCommit := Commit
	Version = "1.2.3"
	Commit = "abc123"
	t.Cleanup(func() {
		Version = previousVersion
		Commit = previousCommit
	})

	var output bytes.Buffer
	if !Show([]string{"/usr/lib/webycp/webycp-server", "--version"}, &output) {
		t.Fatal("expected version argument to be handled")
	}
	if got, want := output.String(), "webycp-server 1.2.3 (abc123)\n"; got != want {
		t.Fatalf("unexpected output: got %q, want %q", got, want)
	}
}

func TestShowIgnoresOtherArguments(t *testing.T) {
	var output bytes.Buffer
	if Show([]string{"webycp-server"}, &output) {
		t.Fatal("expected arguments not to be handled")
	}
	if output.Len() != 0 {
		t.Fatalf("expected no output, got %q", output.String())
	}
}
