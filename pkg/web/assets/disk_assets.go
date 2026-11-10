//go:build !embed
// +build !embed

package assets

import "os"

var EmbeddedAssets = os.DirFS("pkg/web/assets/dist")
