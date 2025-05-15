//go:build !embed
// +build !embed

package assets

import (
	"io/fs"
	"os"
)

type diskFS struct {
	root string
}

func (d diskFS) Open(name string) (fs.File, error) {
	return os.Open(d.root + "/" + name)
}

var EmbeddedAssets fs.FS = diskFS{root: "dist"}
