package migrations

import "embed"

// Files contains all SQL migrations shipped with the backend bootstrap.
//
//go:embed *.sql
var Files embed.FS
