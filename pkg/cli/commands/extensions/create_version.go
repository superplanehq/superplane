package extensions

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/fsnotify/fsnotify"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type CreateVersionCommand struct {
	ExtensionID string
	EntryPoint  string
	Version     string
	Watch       bool
}

func (c *CreateVersionCommand) Execute(ctx core.CommandContext) error {
	if c.Watch && !ctx.Renderer.IsText() {
		return fmt.Errorf("--watch only supports text output")
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}

	entryPoint, err := resolveEntryPoint(projectDir, c.EntryPoint)
	if err != nil {
		return err
	}

	bundle, digest, err := buildExtensionVersionUpload(ctx.Context, projectDir, entryPoint)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.ExtensionAPI.ExtensionsCreateVersion(ctx.Context, c.ExtensionID).
		Body(openapi_client.ExtensionsCreateVersionBody{
			Version: &c.Version,
			Bundle:  &bundle,
			Digest:  &digest,
		}).
		Execute()
	if err != nil {
		return err
	}

	version := response.GetVersion()

	if !c.Watch {
		if !ctx.Renderer.IsText() {
			return ctx.Renderer.Render(version)
		}

		return nil
	}

	_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "Created draft version %s. Watching for changes...\n", c.Version)

	return watchAndUpdateVersion(ctx, c.ExtensionID, projectDir, entryPoint, c.Version)
}

func watchAndUpdateVersion(
	ctx core.CommandContext,
	extensionID string,
	projectDir string,
	entryPoint string,
	versionName string,
) error {
	signalCtx, stop := signal.NotifyContext(ctx.Context, os.Interrupt, syscall.SIGTERM)
	defer stop()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher: %w", err)
	}
	defer watcher.Close()

	watchPaths, err := findWatchPaths(projectDir, entryPoint)
	if err != nil {
		return err
	}

	watchedPaths := make(map[string]struct{}, len(watchPaths))
	for _, path := range watchPaths {
		if err := addWatchPath(watcher, watchedPaths, path); err != nil {
			return err
		}
	}

	debounceDelay := 400 * time.Millisecond
	debounceTimer := time.NewTimer(time.Hour)
	if !debounceTimer.Stop() {
		<-debounceTimer.C
	}

	rebuildPending := false

	for {
		select {
		case <-signalCtx.Done():
			return nil
		case err := <-watcher.Errors:
			if err != nil {
				_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "watch error: %v\n", err)
			}
		case event := <-watcher.Events:
			if err := addWatchPathsForNewDirectory(watcher, watchedPaths, event.Name); err != nil {
				_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "watch error: %v\n", err)
			}

			if !shouldTriggerRebuild(event) {
				continue
			}

			rebuildPending = true
			debounceTimer.Reset(debounceDelay)
		case <-debounceTimer.C:
			if !rebuildPending {
				continue
			}

			rebuildPending = false
			_, _ = fmt.Fprintln(ctx.Cmd.ErrOrStderr(), "Change detected. Rebuilding...")

			bundle, digest, err := buildExtensionVersionUpload(signalCtx, projectDir, entryPoint)
			if err != nil {
				_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "rebuild failed: %v\n", err)
				continue
			}

			_, _, err = ctx.API.ExtensionAPI.ExtensionsUpdateVersion(signalCtx, extensionID, versionName).
				Body(openapi_client.ExtensionsUpdateVersionBody{
					Bundle: &bundle,
					Digest: &digest,
				}).
				Execute()
			if err != nil {
				_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "update failed: %v\n", err)
				continue
			}

			_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "Updated draft version %s.\n", versionName)
		}
	}
}

func resolveEntryPoint(projectDir string, entryPoint string) (string, error) {
	target := strings.TrimSpace(entryPoint)
	if target == "" {
		target = "src/index.ts"
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(projectDir, target)
	}

	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("stat entry point %q: %w", target, err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("entry point %q is a directory", target)
	}

	return target, nil
}

func buildExtensionVersionUpload(
	ctx context.Context,
	projectDir string,
	entryPoint string,
) (string, string, error) {
	artifacts, err := buildExtensionVersionArtifacts(ctx, projectDir, entryPoint)
	if err != nil {
		return "", "", err
	}

	archive, err := buildBundleArchiveForUpload(artifacts)
	if err != nil {
		return "", "", err
	}

	return base64.StdEncoding.EncodeToString(archive), bundleDigest(archive), nil
}

type extensionVersionArtifacts struct {
	RuntimeBundle []byte
	ManifestJSON  []byte
}

func buildExtensionVersionArtifacts(
	ctx context.Context,
	projectDir string,
	entryPoint string,
) (extensionVersionArtifacts, error) {
	sdkEntryPoint, err := findSDKEntryPoint(projectDir)
	if err != nil {
		return extensionVersionArtifacts{}, err
	}

	runtimeEntryPoint, err := findRuntimeEntryPoint(projectDir)
	if err != nil {
		return extensionVersionArtifacts{}, err
	}

	tempDir, err := os.MkdirTemp("", "superplane-extension-*")
	if err != nil {
		return extensionVersionArtifacts{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	runtimeBundlePath := filepath.Join(tempDir, "index.js")
	manifestScriptPath := filepath.Join(tempDir, "manifest.js")

	if err := bundleRuntimeEntry(projectDir, entryPoint, sdkEntryPoint, runtimeEntryPoint, runtimeBundlePath); err != nil {
		return extensionVersionArtifacts{}, err
	}

	if err := bundleManifestEntry(projectDir, entryPoint, sdkEntryPoint, runtimeEntryPoint, manifestScriptPath); err != nil {
		return extensionVersionArtifacts{}, err
	}

	manifestJSON, err := executeNodeScript(ctx, manifestScriptPath)
	if err != nil {
		return extensionVersionArtifacts{}, err
	}

	formattedManifestJSON, err := formatManifestJSON(manifestJSON)
	if err != nil {
		return extensionVersionArtifacts{}, err
	}

	runtimeBundle, err := os.ReadFile(runtimeBundlePath)
	if err != nil {
		return extensionVersionArtifacts{}, fmt.Errorf("read bundled extension: %w", err)
	}

	return extensionVersionArtifacts{
		RuntimeBundle: runtimeBundle,
		ManifestJSON:  formattedManifestJSON,
	}, nil
}

func findSDKEntryPoint(startDir string) (string, error) {
	return findPackageEntryPoint(startDir, filepath.Join("extensions", "sdk", "ts", "src", "index.ts"))
}

func findRuntimeEntryPoint(startDir string) (string, error) {
	return findPackageEntryPoint(startDir, filepath.Join("extensions", "runtime", "ts", "src", "index.ts"))
}

func findPackageEntryPoint(startDir string, relativePath string) (string, error) {
	dir := startDir

	for {
		candidate := filepath.Join(dir, relativePath)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return "", fmt.Errorf("could not find %s from %q", relativePath, startDir)
}

func bundleRuntimeEntry(projectDir, entryPoint, sdkEntryPoint, runtimeEntryPoint, outfile string) error {
	source := fmt.Sprintf(
		"import { createRuntimeModule } from %s;\nimport extension from %s;\nconst runtime = createRuntimeModule(extension);\nexport const manifest = runtime.manifest;\nexport const operations = runtime.operations;\nexport async function run(job) {\n  return await runtime.run(job);\n}\n",
		jsonStringLiteral("@superplanehq/sdk"),
		jsonStringLiteral(entryPoint),
	)

	return runRuntimeEsbuild(projectDir, sdkEntryPoint, runtimeEntryPoint, outfile, source)
}

func bundleManifestEntry(projectDir, entryPoint, sdkEntryPoint, runtimeEntryPoint, outfile string) error {
	source := fmt.Sprintf(
		"import { discoverExtension } from %s;\nimport extension from %s;\nprocess.stdout.write(JSON.stringify(discoverExtension(extension).manifest));\n",
		jsonStringLiteral("@superplanehq/sdk"),
		jsonStringLiteral(entryPoint),
	)

	return runManifestEsbuild(projectDir, sdkEntryPoint, runtimeEntryPoint, outfile, source)
}

func runRuntimeEsbuild(projectDir, sdkEntryPoint, runtimeEntryPoint, outfile, source string) error {
	result := esbuild.Build(esbuild.BuildOptions{
		AbsWorkingDir: projectDir,
		Alias: map[string]string{
			"@superplanehq/runtime": runtimeEntryPoint,
			"@superplanehq/sdk":     sdkEntryPoint,
		},
		Bundle:   true,
		Format:   esbuild.FormatESModule,
		LogLevel: esbuild.LogLevelSilent,
		Outfile:  outfile,
		Platform: esbuild.PlatformNeutral,
		Stdin: &esbuild.StdinOptions{
			Contents:   source,
			Loader:     esbuild.LoaderTS,
			ResolveDir: projectDir,
			Sourcefile: "entry.ts",
		},
		Target: esbuild.ES2022,
		Write:  true,
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("bundle extension: %s", formatEsbuildMessages(result.Errors))
	}

	return nil
}

func runManifestEsbuild(projectDir, sdkEntryPoint, runtimeEntryPoint, outfile, source string) error {
	result := esbuild.Build(esbuild.BuildOptions{
		AbsWorkingDir: projectDir,
		Alias: map[string]string{
			"@superplanehq/runtime": runtimeEntryPoint,
			"@superplanehq/sdk":     sdkEntryPoint,
		},
		Bundle:   true,
		Format:   esbuild.FormatCommonJS,
		LogLevel: esbuild.LogLevelSilent,
		Outfile:  outfile,
		Platform: esbuild.PlatformNode,
		Stdin: &esbuild.StdinOptions{
			Contents:   source,
			Loader:     esbuild.LoaderTS,
			ResolveDir: projectDir,
			Sourcefile: "entry.ts",
		},
		Target: esbuild.ES2022,
		Write:  true,
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("bundle extension: %s", formatEsbuildMessages(result.Errors))
	}

	return nil
}

func executeNodeScript(ctx context.Context, scriptPath string) ([]byte, error) {
	command := exec.CommandContext(ctx, "node", scriptPath)
	output, err := command.Output()
	if err == nil {
		return output, nil
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		stderr := strings.TrimSpace(string(exitError.Stderr))
		if stderr != "" {
			return nil, fmt.Errorf("run node manifest extractor: %s", stderr)
		}
	}

	return nil, fmt.Errorf("run node manifest extractor: %w", err)
}

func buildBundleArchiveForUpload(artifacts extensionVersionArtifacts) ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "superplane-extension-dist-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dist dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	distDir := filepath.Join(tempDir, "dist")
	if err := writeExtensionVersionArtifacts(distDir, artifacts); err != nil {
		return nil, err
	}

	return createBundleArchiveFromDirectory(distDir)
}

func writeExtensionVersionArtifacts(destination string, artifacts extensionVersionArtifacts) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create destination directory %q: %w", destination, err)
	}

	bundlePath := filepath.Join(destination, "bundle.js")
	if err := os.WriteFile(bundlePath, artifacts.RuntimeBundle, 0o600); err != nil {
		return fmt.Errorf("write bundle to %q: %w", bundlePath, err)
	}

	manifestPath := filepath.Join(destination, "manifest.json")
	if err := os.WriteFile(manifestPath, artifacts.ManifestJSON, 0o600); err != nil {
		return fmt.Errorf("write manifest to %q: %w", manifestPath, err)
	}

	return nil
}

func createBundleArchiveFromDirectory(directory string) ([]byte, error) {
	entries := make([]string, 0)
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		entries = append(entries, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan bundle directory: %w", err)
	}

	sort.Strings(entries)

	buffer := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	root := filepath.Dir(directory)
	for _, path := range entries {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat bundle entry %q: %w", path, err)
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return nil, fmt.Errorf("compute bundle path for %q: %w", path, err)
		}

		header := &tar.Header{
			Mode:    0o644,
			ModTime: time.Unix(0, 0).UTC(),
			Name:    filepath.ToSlash(relativePath),
			Size:    info.Size(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("write tar header for %q: %w", relativePath, err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read bundle entry %q: %w", path, err)
		}

		if _, err := tarWriter.Write(contents); err != nil {
			return nil, fmt.Errorf("write tar contents for %q: %w", relativePath, err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("close tar archive: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close gzip archive: %w", err)
	}

	return buffer.Bytes(), nil
}

func bundleDigest(bundle []byte) string {
	sum := sha256.Sum256(bundle)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func formatEsbuildMessages(messages []esbuild.Message) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(message.Text)
		if text == "" {
			continue
		}

		if message.Location == nil || message.Location.File == "" {
			parts = append(parts, text)
			continue
		}

		parts = append(parts, fmt.Sprintf("%s:%d:%d: %s", message.Location.File, message.Location.Line, message.Location.Column, text))
	}

	return strings.Join(parts, "; ")
}

func jsonStringLiteral(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func findWatchPaths(projectDir string, entryPoint string) ([]string, error) {
	paths := []string{filepath.Dir(entryPoint)}
	for _, candidate := range []string{"integrations", "components", "triggers"} {
		path := filepath.Join(projectDir, candidate)
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			paths = append(paths, path)
		}
	}

	seen := make(map[string]struct{}, len(paths))
	uniquePaths := make([]string, 0, len(paths))
	for _, path := range paths {
		walkPaths, err := collectWatchDirectories(path)
		if err != nil {
			return nil, err
		}

		for _, walkPath := range walkPaths {
			cleanPath := filepath.Clean(walkPath)
			if _, ok := seen[cleanPath]; ok {
				continue
			}

			seen[cleanPath] = struct{}{}
			uniquePaths = append(uniquePaths, cleanPath)
		}
	}

	return uniquePaths, nil
}

func collectWatchDirectories(root string) ([]string, error) {
	directories := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !entry.IsDir() {
			return nil
		}

		directories = append(directories, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan watch paths from %q: %w", root, err)
	}

	return directories, nil
}

func addWatchPath(watcher *fsnotify.Watcher, watchedPaths map[string]struct{}, path string) error {
	cleanPath := filepath.Clean(path)
	if _, ok := watchedPaths[cleanPath]; ok {
		return nil
	}

	if err := watcher.Add(cleanPath); err != nil {
		return fmt.Errorf("watch %q: %w", cleanPath, err)
	}

	watchedPaths[cleanPath] = struct{}{}
	return nil
}

func addWatchPathsForNewDirectory(watcher *fsnotify.Watcher, watchedPaths map[string]struct{}, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("stat watch path %q: %w", path, err)
	}

	if !info.IsDir() {
		return nil
	}

	paths, err := collectWatchDirectories(path)
	if err != nil {
		return err
	}

	for _, watchPath := range paths {
		if err := addWatchPath(watcher, watchedPaths, watchPath); err != nil {
			return err
		}
	}

	return nil
}

func shouldTriggerRebuild(event fsnotify.Event) bool {
	return event.Op&fsnotify.Create == fsnotify.Create ||
		event.Op&fsnotify.Write == fsnotify.Write ||
		event.Op&fsnotify.Remove == fsnotify.Remove ||
		event.Op&fsnotify.Rename == fsnotify.Rename
}
