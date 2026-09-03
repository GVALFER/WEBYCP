package buildinfo

import (
	"fmt"
	"io"
	"path/filepath"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func Show(args []string, output io.Writer) bool {
	if len(args) != 2 || args[1] != "--version" {
		return false
	}

	fmt.Fprintf(output, "%s %s (%s)\n", filepath.Base(args[0]), Version, Commit)
	return true
}
