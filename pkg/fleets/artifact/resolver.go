package artifact

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

const (
	maxManifestBytes        = 1 << 20
	maxArtifactBytes        = 512 << 20
	supportedRunnerProtocol = "runner/v1"
)

var exactVersionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

type Artifact struct {
	Version         string
	OperatingSystem string
	Architecture    string
	URL             string
	SHA256          string
	Signature       string
	ProtocolVersion string
}

type manifest struct {
	Version          string             `json:"version"`
	ProtocolVersion  string             `json:"protocol_version"`
	SourceRepository string             `json:"source_repository,omitempty"`
	SourceCommit     string             `json:"source_commit,omitempty"`
	Artifacts        []manifestArtifact `json:"artifacts"`
}

type manifestArtifact struct {
	OperatingSystem string `json:"operating_system"`
	Architecture    string `json:"architecture"`
	URL             string `json:"url"`
	SHA256          string `json:"sha256"`
	Signature       string `json:"signature"`
}

type Resolver struct {
	manifestURLTemplate string
	publicKey           ed25519.PublicKey
	httpClient          *http.Client

	mu    sync.RWMutex
	cache map[string]Artifact
}

func NewResolver(
	manifestURLTemplate, encodedPublicKey string,
	httpClient *http.Client,
) (*Resolver, error) {
	template := strings.TrimSpace(manifestURLTemplate)
	if !strings.Contains(template, "{version}") {
		return nil, fmt.Errorf("runner manifest URL template must contain {version}")
	}
	if strings.Contains(strings.ToLower(template), "latest") {
		return nil, fmt.Errorf("runner manifest URL template must not use latest")
	}
	publicKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedPublicKey))
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("runner artifact signing public key must be a base64 Ed25519 public key")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Resolver{
		manifestURLTemplate: template,
		publicKey:           ed25519.PublicKey(publicKey),
		httpClient:          httpClient,
		cache:               map[string]Artifact{},
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
		return Artifact{}, fmt.Errorf("runner version %q is not an exact semantic version", version)
	}
	if operatingSystem == "" || architecture == "" {
		return Artifact{}, fmt.Errorf("runner operating system and architecture are required")
	}

	cacheKey := strings.Join([]string{version, operatingSystem, architecture}, "\x00")
	r.mu.RLock()
	cached, ok := r.cache[cacheKey]
	r.mu.RUnlock()
	if ok {
		return cached, nil
	}

	resolved, err := r.resolve(ctx, version, operatingSystem, architecture)
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
	version, operatingSystem, architecture string,
) (Artifact, error) {
	manifestURL := strings.ReplaceAll(r.manifestURLTemplate, "{version}", url.PathEscape(version))
	var release manifest
	if err := r.readJSON(ctx, manifestURL, maxManifestBytes, &release); err != nil {
		return Artifact{}, fmt.Errorf("read runner release manifest: %w", err)
	}
	if release.Version != version {
		return Artifact{}, fmt.Errorf(
			"runner release manifest version %q does not match requested version %q",
			release.Version,
			version,
		)
	}
	if release.ProtocolVersion != supportedRunnerProtocol {
		return Artifact{}, fmt.Errorf(
			"runner release %s uses unsupported protocol %q",
			version,
			release.ProtocolVersion,
		)
	}

	for _, candidate := range release.Artifacts {
		if strings.EqualFold(candidate.OperatingSystem, operatingSystem) &&
			strings.EqualFold(candidate.Architecture, architecture) {
			return r.verify(ctx, release, candidate)
		}
	}
	return Artifact{}, fmt.Errorf(
		"runner release %s has no artifact for %s/%s",
		version,
		operatingSystem,
		architecture,
	)
}

func (r *Resolver) verify(
	ctx context.Context,
	release manifest,
	candidate manifestArtifact,
) (Artifact, error) {
	if err := validatePublicURL(candidate.URL); err != nil {
		return Artifact{}, fmt.Errorf("runner artifact URL: %w", err)
	}
	if err := validateVersionedArtifactURL(candidate.URL, release.Version); err != nil {
		return Artifact{}, err
	}
	expectedDigest, err := hex.DecodeString(strings.ToLower(strings.TrimSpace(candidate.SHA256)))
	if err != nil || len(expectedDigest) != sha256.Size {
		return Artifact{}, fmt.Errorf("runner artifact SHA-256 is invalid")
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(candidate.Signature))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return Artifact{}, fmt.Errorf("runner artifact signature is invalid")
	}
	if !ed25519.Verify(r.publicKey, expectedDigest, signature) {
		return Artifact{}, fmt.Errorf("runner artifact signature verification failed")
	}

	actualDigest, err := r.hashURL(ctx, candidate.URL)
	if err != nil {
		return Artifact{}, fmt.Errorf("verify runner artifact: %w", err)
	}
	if !equalBytes(actualDigest, expectedDigest) {
		return Artifact{}, fmt.Errorf("runner artifact SHA-256 does not match release manifest")
	}

	return Artifact{
		Version:         release.Version,
		OperatingSystem: strings.ToLower(candidate.OperatingSystem),
		Architecture:    strings.ToLower(candidate.Architecture),
		URL:             candidate.URL,
		SHA256:          hex.EncodeToString(expectedDigest),
		Signature:       candidate.Signature,
		ProtocolVersion: release.ProtocolVersion,
	}, nil
}

func (r *Resolver) readJSON(ctx context.Context, endpoint string, limit int64, target any) error {
	response, err := r.get(ctx, endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return err
	}
	if int64(len(content)) > limit {
		return fmt.Errorf("response exceeds %d bytes", limit)
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("response contains multiple JSON values")
	}
	return nil
}

func (r *Resolver) hashURL(ctx context.Context, endpoint string) ([]byte, error) {
	response, err := r.get(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	hasher := sha256.New()
	written, err := io.Copy(hasher, io.LimitReader(response.Body, maxArtifactBytes+1))
	if err != nil {
		return nil, err
	}
	if written > maxArtifactBytes {
		return nil, fmt.Errorf("runner artifact exceeds %d bytes", maxArtifactBytes)
	}
	return hasher.Sum(nil), nil
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

func validateVersionedArtifactURL(raw, version string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("runner artifact URL is invalid: %w", err)
	}
	if strings.Contains(strings.ToLower(parsed.Path), "latest") {
		return fmt.Errorf("runner artifact URL must not use latest")
	}
	version = strings.TrimPrefix(version, "v")
	for _, segment := range strings.Split(strings.Trim(parsed.Path, "/"), "/") {
		if segment == version || segment == "v"+version {
			return nil
		}
	}
	return fmt.Errorf("runner artifact URL does not contain exact version %q", version)
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var different byte
	for i := range left {
		different |= left[i] ^ right[i]
	}
	return different == 0
}
