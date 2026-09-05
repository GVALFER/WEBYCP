package account

import (
	"context"
	"errors"
	"fmt"

	agentruntime "github.com/GVALFER/WEBYCP/internal/agent/runtime"
)

type Manager struct {
	*Linux
	runtime agentruntime.Cleaner
	ftp     FileAccess
}

type FileAccess interface {
	Disable(context.Context, string, string) error
	Enable(context.Context, string, string) error
	Delete(context.Context, string, string) error
}

func New(linux *Linux, runtime agentruntime.Cleaner, ftp FileAccess) *Manager {
	return &Manager{Linux: linux, runtime: runtime, ftp: ftp}
}

func (m *Manager) Disable(ctx context.Context, accountID, systemUser string) error {
	return errors.Join(m.Linux.Disable(ctx, accountID, systemUser), m.ftp.Disable(ctx, accountID, systemUser))
}

func (m *Manager) Enable(ctx context.Context, accountID, systemUser string) error {
	if err := m.Linux.Enable(ctx, accountID, systemUser); err != nil {
		return err
	}
	if err := m.ftp.Enable(ctx, accountID, systemUser); err != nil {
		return errors.Join(
			fmt.Errorf("enable account FTP access: %w", err),
			m.ftp.Disable(ctx, accountID, systemUser),
			m.Linux.Disable(ctx, accountID, systemUser),
		)
	}
	return nil
}

func (m *Manager) Delete(ctx context.Context, accountID, systemUser string) error {
	if err := m.ftp.Delete(ctx, accountID, systemUser); err != nil {
		return fmt.Errorf("delete account FTP access: %w", err)
	}
	if err := m.runtime.Delete(ctx, accountID); err != nil {
		return fmt.Errorf("delete account runtime: %w", err)
	}
	return m.Linux.Delete(ctx, accountID, systemUser)
}
