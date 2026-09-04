package account

import (
	"context"
	"fmt"

	agentruntime "github.com/GVALFER/WEBYCP/internal/agent/runtime"
)

type Manager struct {
	*Linux
	runtime agentruntime.Cleaner
}

func New(linux *Linux, runtime agentruntime.Cleaner) *Manager {
	return &Manager{Linux: linux, runtime: runtime}
}

func (m *Manager) Delete(ctx context.Context, accountID, systemUser string) error {
	if err := m.runtime.Delete(ctx, accountID); err != nil {
		return fmt.Errorf("delete account runtime: %w", err)
	}
	return m.Linux.Delete(ctx, accountID, systemUser)
}
