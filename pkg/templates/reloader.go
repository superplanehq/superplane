package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/registry"
)

var templateReloadLock sync.Mutex

func StartTemplateReloader(registry *registry.Registry) {
	dir, ok := templateDir()
	if !ok {
		return
	}

	initialFingerprint, err := templateFingerprint(dir)
	if err != nil {
		log.Printf("template reloader: failed to read templates: %v", err)
	}

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		lastFingerprint := initialFingerprint

		for range ticker.C {
			fingerprint, err := templateFingerprint(dir)
			if err != nil {
				log.Printf("template reloader: failed to read templates: %v", err)
				continue
			}

			if fingerprint == lastFingerprint {
				continue
			}

			templateReloadLock.Lock()
			err = SeedTemplates(registry)
			templateReloadLock.Unlock()
			if err != nil {
				log.Printf("template reloader: failed to seed templates: %v", err)
			} else {
				log.Printf("template reloader: templates re-seeded")
				lastFingerprint = fingerprint
			}
		}
	}()
}

func templateFingerprint(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		files = append(files, entry.Name())
	}

	sort.Strings(files)

	hasher := sha256.New()
	for _, name := range files {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			return "", err
		}
		_, _ = hasher.Write([]byte(name))
		_, _ = hasher.Write([]byte(info.ModTime().UTC().Format(time.RFC3339Nano)))
		_, _ = hasher.Write([]byte(fmt.Sprintf("%d", info.Size())))
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
