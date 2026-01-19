package templates

import (
	"io/fs"
	"os"
)

func templateDir() (string, bool) {
	if os.Getenv("APP_ENV") != "development" {
		return "", false
	}

	dir := os.Getenv("TEMPLATE_DIR")
	if dir == "" {
		return "", false
	}

	return dir, true
}

func templateFS() (fs.FS, string, error) {
	if dir, ok := templateDir(); ok {
		return os.DirFS(dir), ".", nil
	}

	return templateAssets, "templates", nil
}
