//go:build !embed
// +build !embed

package assets

import (
	"io/fs"
	"os"

	log "github.com/sirupsen/logrus"
)

type diskFS struct {
	root string
}

func (d diskFS) Open(name string) (fs.File, error) {
	log.Infof("Opening asset %s/%s", d.root, name)
	return os.Open(d.root + "/" + name)
}

var EmbeddedAssets fs.FS = diskFS{root: "pkg/web/assets/dist"}
