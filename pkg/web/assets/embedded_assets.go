//go:build embed
// +build embed

package assets

import (
	"embed"
	_ "embed"
)

//go:embed dist/*
var EmbeddedAssets embed.FS
