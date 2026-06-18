package installation

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
)

// Limits guard the tarball extraction so a malicious or oversized repository
// cannot exhaust worker memory or the seed-files table. Conservative values:
// most legitimate SuperPlane app repos are small.
const (
	maxRepositoryFileCount     = 200
	maxRepositoryFileSizeBytes = 1 << 20  // 1 MiB per file
	maxRepositoryTotalSize     = 10 << 20 // 10 MiB total
	maxTarballDownloadSize     = 20 << 20 // 20 MiB compressed tarball
)

// RepositoryFile is a file extracted from a GitHub repository tarball, ready to
// be seeded into the canvas git repository.
type RepositoryFile struct {
	Path    string
	Content []byte
}

// excludedRepositoryFiles are repo-relative paths that the install flow either
// handles separately (canvas/console spec files) or must not copy
// (params.json carries install-time configuration, not runtime files).
var excludedRepositoryFiles = map[string]struct{}{
	paramsFileName:  {},
	canvasFileName:  {},
	consoleFileName: {},
}

// tarballHTTPGet is overridable in tests so we can serve fixture tarballs
// without depending on codeload.github.com.
var tarballHTTPGet = func(rawURL string) (*http.Response, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	return client.Get(rawURL)
}

// FetchRepositoryFiles downloads the GitHub repo tarball at the given ref and
// returns every regular file except the spec files (canvas.yaml/console.yaml)
// and params.json. Paths are validated with the same rules as user-supplied
// repository writes so reserved or escaping paths cannot reach the git
// provider through the seed-files path.
func FetchRepositoryFiles(repo *Repository, ref string) ([]RepositoryFile, error) {
	if ref == "" {
		return nil, fmt.Errorf("repository file fetch requires a resolved ref")
	}

	body, err := fetchTarball(tarballURL(repo, ref))
	if err != nil {
		if errors.Is(err, errFileNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return extractRepositoryFiles(body)
}

func tarballURL(repo *Repository, ref string) string {
	return fmt.Sprintf(
		"https://codeload.github.com/%s/%s/tar.gz/%s",
		repo.Owner,
		repo.Name,
		ref,
	)
}

func fetchTarball(rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if parsed.Scheme != "https" || parsed.Host != "codeload.github.com" {
		return nil, fmt.Errorf("unsupported tarball host %q", parsed.Host)
	}

	response, err := tarballHTTPGet(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s not found: %w", rawURL, errFileNotFound)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: unexpected status %d", rawURL, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTarballDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rawURL, err)
	}
	if len(body) > maxTarballDownloadSize {
		return nil, fmt.Errorf("tarball %s exceeds maximum size of %d bytes", rawURL, maxTarballDownloadSize)
	}

	return body, nil
}

// extractRepositoryFiles ungzips and untars the codeload payload, dropping the
// "{repo}-{ref}/" top-level directory that GitHub prepends and skipping
// directories, symlinks, the .git folder, and the excluded files above.
func extractRepositoryFiles(body []byte) ([]RepositoryFile, error) {
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gunzip tarball: %w", err)
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	files := make([]RepositoryFile, 0, 32)
	totalSize := int64(0)

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tarball: %w", err)
		}

		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}

		relPath, ok := stripTopLevelDir(header.Name)
		if !ok || relPath == "" {
			continue
		}

		if shouldSkipExtractedPath(relPath) {
			continue
		}

		normalized, err := gitprovider.ValidateUserPath(relPath)
		if err != nil {
			// Skip files that would be rejected at commit time (reserved
			// paths, traversal attempts) rather than fail the whole install.
			continue
		}

		if header.Size > maxRepositoryFileSizeBytes {
			return nil, fmt.Errorf("repository file %q exceeds maximum size of %d bytes", normalized, maxRepositoryFileSizeBytes)
		}

		content, err := io.ReadAll(io.LimitReader(reader, maxRepositoryFileSizeBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read repository file %q: %w", normalized, err)
		}
		if int64(len(content)) > maxRepositoryFileSizeBytes {
			return nil, fmt.Errorf("repository file %q exceeds maximum size of %d bytes", normalized, maxRepositoryFileSizeBytes)
		}

		totalSize += int64(len(content))
		if totalSize > maxRepositoryTotalSize {
			return nil, fmt.Errorf("repository total size exceeds maximum of %d bytes", maxRepositoryTotalSize)
		}

		files = append(files, RepositoryFile{Path: normalized, Content: content})
		if len(files) > maxRepositoryFileCount {
			return nil, fmt.Errorf("repository file count exceeds maximum of %d", maxRepositoryFileCount)
		}
	}

	return files, nil
}

// stripTopLevelDir removes the "{repo}-{ref}/" prefix codeload prepends to
// every entry. Entries that are the top-level directory itself or have no
// slash are dropped.
func stripTopLevelDir(name string) (string, bool) {
	name = strings.TrimLeft(name, "/")
	idx := strings.Index(name, "/")
	if idx < 0 {
		return "", false
	}
	return name[idx+1:], true
}

func shouldSkipExtractedPath(relPath string) bool {
	cleaned := path.Clean(relPath)

	if cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") {
		return true
	}

	if _, ok := excludedRepositoryFiles[cleaned]; ok {
		return true
	}

	return false
}
