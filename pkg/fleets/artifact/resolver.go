package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

const maxChecksumsBytes = 1 << 20

var exactVersionPattern = regexp.MustCompile(
	`^(?:sha:[0-9a-f]{40}|v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)$`,
)

type Artifact struct {
	Version         string
	OperatingSystem string
	Architecture    string
	URL             string
	SHA256          string
}

type Resolver struct {
	baseURL    string
	httpClient *http.Client

	mu    sync.RWMutex
	cache map[string]Artifact
}

func NewResolver(baseURL string, httpClient *http.Client) (*Resolver, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if err := validatePublicURL(baseURL); err != nil {
		return nil, fmt.Errorf("runner release base URL: %w", err)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("runner release base URL: %w", err)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("runner release base URL must not contain a query or fragment")
	}
	if strings.Contains(strings.ToLower(parsed.Path), "latest") {
		return nil, fmt.Errorf("runner release base URL must not use latest")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Resolver{
		baseURL:    baseURL,
		httpClient: httpClient,
		cache:      map[string]Artifact{},
	}, nil
}

func (r *Resolver) Resolve(
	ctx context.Context,
	version, operatingSystem, architecture string,
) (Artifact, error) {
	version = strings.TrimSpace(version)
	operatingSystem = strings.ToLower(strings.TrimSpace(operatingSystem))
	architecture = strings.ToLower(strings.TrimSpace(architecture))
	if !exactVersionPattern.MatchString(version) {
		return Artifact{}, fmt.Errorf(
			"runner version %q must be a v-prefixed semantic version or sha:<40 lowercase hex characters>",
			version,
		)
	}
	filename, err := artifactFilename(operatingSystem, architecture)
	if err != nil {
		return Artifact{}, err
	}

	cacheKey := strings.Join([]string{version, operatingSystem, architecture}, "\x00")
	r.mu.RLock()
	cached, ok := r.cache[cacheKey]
	r.mu.RUnlock()
	if ok {
		return cached, nil
	}

	resolved, err := r.resolve(ctx, version, operatingSystem, architecture, filename)
	if err != nil {
		return Artifact{}, err
	}
	r.mu.Lock()
	r.cache[cacheKey] = resolved
	r.mu.Unlock()
	return resolved, nil
}

func (r *Resolver) resolve(
	ctx context.Context,
	version, operatingSystem, architecture, filename string,
) (Artifact, error) {
	releaseURL := r.baseURL + "/" + url.PathEscape(version)
	checksum, err := r.readChecksum(
		ctx,
		releaseURL+"/checksums.txt",
		filename,
	)
	if err != nil {
		return Artifact{}, fmt.Errorf(
			"resolve runner release %s for %s/%s: %w",
			version,
			operatingSystem,
			architecture,
			err,
		)
	}

	return Artifact{
		Version:         version,
		OperatingSystem: operatingSystem,
		Architecture:    architecture,
		URL:             releaseURL + "/" + filename,
		SHA256:          checksum,
	}, nil
}

func (r *Resolver) readChecksum(
	ctx context.Context,
	endpoint, filename string,
) (string, error) {
	response, err := r.get(ctx, endpoint)
	if err != nil {
		return "", fmt.Errorf("read checksums: %w", err)
	}
	defer response.Body.Close()

	content, err := io.ReadAll(io.LimitReader(response.Body, maxChecksumsBytes+1))
	if err != nil {
		return "", fmt.Errorf("read checksums: %w", err)
	}
	if len(content) > maxChecksumsBytes {
		return "", fmt.Errorf("checksums exceed %d bytes", maxChecksumsBytes)
	}

	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 ||
			strings.TrimPrefix(fields[1], "*") != filename {
			continue
		}
		checksum := strings.ToLower(fields[0])
		decoded, decodeErr := hex.DecodeString(checksum)
		if decodeErr != nil || len(decoded) != sha256.Size {
			return "", fmt.Errorf("checksum for %s is invalid", filename)
		}
		return checksum, nil
	}
	return "", fmt.Errorf("checksums do not contain %s", filename)
}

func (r *Resolver) get(ctx context.Context, endpoint string) (*http.Response, error) {
	if err := validatePublicURL(endpoint); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := r.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if err := validatePublicURL(response.Request.URL.String()); err != nil {
		_ = response.Body.Close()
		return nil, fmt.Errorf("redirected URL: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		return nil, fmt.Errorf("GET %s returned %s", endpoint, response.Status)
	}
	return response, nil
}

func validatePublicURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if parsed.User != nil || parsed.Host == "" {
		return fmt.Errorf("must be an absolute URL without user information")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	host := parsed.Hostname()
	if parsed.Scheme == "http" && (host == "localhost" || net.ParseIP(host).IsLoopback()) {
		return nil
	}
	return fmt.Errorf("must use HTTPS")
}

func artifactFilename(operatingSystem, architecture string) (string, error) {
	switch {
	case operatingSystem != "linux":
		return "", fmt.Errorf(
			"runner release does not support operating system %q",
			operatingSystem,
		)
	case architecture != "amd64" && architecture != "arm64":
		return "", fmt.Errorf(
			"runner release does not support architecture %q",
			architecture,
		)
	default:
		return fmt.Sprintf(
			"runner-%s-%s.tar.gz",
			operatingSystem,
			architecture,
		), nil
	}
}
