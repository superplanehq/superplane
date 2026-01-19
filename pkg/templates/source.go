package templates

import (
	"fmt"
	"io/fs"
	"os"
)

func templateDir() (fs.FS, error) {
	dir := os.Getenv("TEMPLATE_DIR")
	if dir == "" {
		return nil, fmt.Errorf("TEMPLATE_DIR is not set")
	}

	return os.DirFS(dir + "/canvases"), nil
}
