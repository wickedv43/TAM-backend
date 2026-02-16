//go:build tools
// +build tools

// This file declares dependencies on tool binaries that are used in the build process.
// See: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package tools

import (
	_ "github.com/abice/go-enum"
	_ "github.com/pressly/goose/v3/cmd/goose"
)
