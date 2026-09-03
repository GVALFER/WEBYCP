package execx

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const maxOutput = 4 << 10

func Run(ctx context.Context, name string, args ...string) error {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err == nil {
		return nil
	}
	if len(output) > maxOutput {
		output = output[:maxOutput]
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, message)
}

func Write(ctx context.Context, output io.Writer, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout = output
	var stderr strings.Builder
	command.Stderr = &limitWriter{writer: &stderr, remaining: maxOutput}
	if err := command.Run(); err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}
	return nil
}

func Input(ctx context.Context, input io.Reader, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = input
	var output strings.Builder
	command.Stdout = &limitWriter{writer: &output, remaining: maxOutput}
	command.Stderr = &limitWriter{writer: &output, remaining: maxOutput}
	if err := command.Run(); err != nil {
		if message := strings.TrimSpace(output.String()); message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}
	return nil
}

type limitWriter struct {
	writer    io.Writer
	remaining int
}

func (w *limitWriter) Write(value []byte) (int, error) {
	written := len(value)
	if w.remaining <= 0 {
		return written, nil
	}
	if len(value) > w.remaining {
		value = value[:w.remaining]
	}
	count, err := w.writer.Write(value)
	w.remaining -= count
	if err != nil {
		return count, err
	}
	return written, nil
}
