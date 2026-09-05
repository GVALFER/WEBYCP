package pureftpd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GVALFER/WEBYCP/internal/execx"
)

func disconnect(ctx context.Context, uid int) error {
	if uid <= 0 {
		return fmt.Errorf("invalid FTP session identity")
	}
	var output bytes.Buffer
	if err := execx.Write(ctx, &output, "/usr/sbin/pure-ftpwho", "-s", "-n"); err != nil {
		return fmt.Errorf("inspect active FTP sessions")
	}
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if line == "" {
			continue
		}
		pid, _, ok := strings.Cut(line, "|")
		id, err := strconv.Atoi(pid)
		if !ok || err != nil || id <= 1 {
			return fmt.Errorf("invalid FTP session report")
		}
		if err := stopSession(id, uid); err != nil {
			return err
		}
	}
	return nil
}

func stopSession(pid, uid int) error {
	// On Linux, FindProcess binds a pidfd. Verify the executable and effective
	// UID before signalling that handle; never trust a PID from a report alone.
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	defer process.Release()
	root := filepath.Join("/proc", strconv.Itoa(pid))
	executable, err := os.Readlink(filepath.Join(root, "exe"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect FTP session executable: %w", err)
	}
	if strings.TrimSuffix(executable, " (deleted)") != "/usr/sbin/pure-ftpd" {
		return nil
	}
	status, err := os.ReadFile(filepath.Join(root, "status"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect FTP session owner: %w", err)
	}
	if !sessionOwner(string(status), uid) {
		return nil
	}
	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("terminate FTP session: %w", err)
	}
	return nil
}

func sessionOwner(status string, uid int) bool {
	if uid <= 0 {
		return false
	}
	for _, line := range strings.Split(status, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 5 && fields[0] == "Uid:" {
			return fields[2] == strconv.Itoa(uid)
		}
	}
	return false
}
