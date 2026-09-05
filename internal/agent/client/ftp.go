package client

import (
	"context"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/agent/ftp"
	agentapi "github.com/GVALFER/WEBYCP/internal/agent/protocol"
)

func (c *Client) SyncFTP(ctx context.Context, socket, accountID, systemUser string, entries []ftp.Entry) error {
	values := make([]agentapi.FTPEntry, 0, len(entries))
	for _, entry := range entries {
		values = append(values, agentapi.FTPEntry{Id: entry.ID, Username: entry.Username, PasswordHash: entry.PasswordHash, Enabled: entry.Enabled})
	}
	return c.sendJSON(ctx, socket, http.MethodPut, "/agent/v1/ftp-accounts", agentapi.SyncFTPRequest{
		AccountId: accountID, SystemUser: systemUser, Entries: values,
	}, "FTP access synchronization")
}
