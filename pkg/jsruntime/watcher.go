package jsruntime

import (
	"context"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/registry"
)

const defaultWatchInterval = 5 * time.Second

type fileState struct {
	modTime time.Time
}

// Watcher polls a directory for JS component file changes and keeps the registry in sync.
type Watcher struct {
	dir      string
	interval time.Duration
	runtime  *Runtime
	registry *registry.Registry
	logger   *log.Entry
	known    map[string]fileState
}

func NewWatcher(dir string, interval time.Duration, rt *Runtime, reg *registry.Registry) *Watcher {
	if interval == 0 {
		interval = defaultWatchInterval
	}

	return &Watcher{
		dir:      dir,
		interval: interval,
		runtime:  rt,
		registry: reg,
		logger:   log.WithField("worker", "JSComponentWatcher"),
		known:    make(map[string]fileState),
	}
}

// SetInitialState records the current state of known files so the first tick only detects
// actual changes rather than re-registering everything.
func (w *Watcher) SetInitialState(registryNames []string) {
	for _, name := range registryNames {
		baseName := strings.TrimPrefix(name, jsComponentPrefix)
		filename := baseName + ".js"

		info, err := os.Stat(w.dir + "/" + filename)
		if err != nil {
			continue
		}

		w.known[filename] = fileState{modTime: info.ModTime()}
	}
}

// Start begins polling the directory for changes. It blocks until the context is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	w.logger.Infof("Watching %s for JS component changes (interval: %s)", w.dir, w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("JS component watcher stopped")
			return
		case <-ticker.C:
			w.poll()
		}
	}
}

func (w *Watcher) poll() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		w.logger.WithError(err).Error("Failed to read JS components directory")
		return
	}

	currentFiles := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}

		filename := entry.Name()
		currentFiles[filename] = true

		info, err := entry.Info()
		if err != nil {
			w.logger.WithError(err).Errorf("Failed to stat %s", filename)
			continue
		}

		prev, exists := w.known[filename]

		if !exists {
			w.handleNewFile(filename)
			w.known[filename] = fileState{modTime: info.ModTime()}
			continue
		}

		if info.ModTime().After(prev.modTime) {
			w.handleModifiedFile(filename)
			w.known[filename] = fileState{modTime: info.ModTime()}
		}
	}

	for filename := range w.known {
		if !currentFiles[filename] {
			w.handleDeletedFile(filename)
			delete(w.known, filename)
		}
	}
}

func (w *Watcher) handleNewFile(filename string) {
	registryName, err := loadComponentFile(w.dir, filename, w.runtime, w.registry)
	if err != nil {
		w.logger.WithError(err).Errorf("Failed to load new JS component %s", filename)
		return
	}

	w.logger.Infof("Loaded new JS component: %s", registryName)
}

func (w *Watcher) handleModifiedFile(filename string) {
	registryName, err := loadComponentFile(w.dir, filename, w.runtime, w.registry)
	if err != nil {
		w.logger.WithError(err).Errorf(
			"Failed to reload JS component %s, keeping previous version", filename,
		)
		return
	}

	w.logger.Infof("Reloaded JS component: %s", registryName)
}

func (w *Watcher) handleDeletedFile(filename string) {
	registryName := RegistryNameFromFilename(filename)
	w.registry.UnregisterComponent(registryName)
	w.logger.Infof("Unregistered deleted JS component: %s", registryName)
}
