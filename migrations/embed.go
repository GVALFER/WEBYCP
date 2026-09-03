package migrations

import "embed"

// Files contains the forward-only database migrations shipped with WEBYCP.
//
//go:embed *.sql
var Files embed.FS

