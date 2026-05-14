//go:build !embed
// +build !embed

package assets

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type diskFS struct {
	root string
}

func (d diskFS) Open(name string) (fs.File, error) {
	log.Infof("Opening asset %s/%s", d.root, name)

	clean := path.Clean("/" + name)
	full := filepath.Join(d.root, clean)

	rel, err := filepath.Rel(d.root, full)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(rel, "..") {
		return nil, fs.ErrPermission
	}
	return os.Open(full)
}

var EmbeddedAssets fs.FS = diskFS{root: "pkg/web/assets/dist"}
