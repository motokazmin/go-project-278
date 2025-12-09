//go:build tools

package tools

import (
	// Blank import to pin goose in go.mod/vendor for offline builds.
	_ "github.com/pressly/goose/v3/cmd/goose"
)
