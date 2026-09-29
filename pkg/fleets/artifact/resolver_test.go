package artifact

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestResolveRequiresExactVersionAndVerifiesArtifact(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	binary := []byte("runner binary")
	digest := sha256.Sum256(binary)
	signature := ed25519.Sign(privateKey, digest[:])
	var artifactReads atomic.Int32

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/runner/v1.2.3/manifest.json":
			fmt.Fprintf(w, `{
				"version":"1.2.3",
				"protocol_version":"runner/v1",
				"source_repository":"https://github.com/superplanehq/superplane",
				"source_commit":"0123456789abcdef",
				"artifacts":[{
					"operating_system":"linux",
					"architecture":"arm64",
					"url":%q,
					"sha256":%q,
					"signature":%q
				}]
			}`,
				server.URL+"/runner/v1.2.3/runner-linux-arm64",
				hex.EncodeToString(digest[:]),
				base64.StdEncoding.EncodeToString(signature),
			)
		case "/runner/v1.2.3/runner-linux-arm64":
			artifactReads.Add(1)
			_, _ = w.Write(binary)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resolver, err := NewResolver(
		server.URL+"/runner/v{version}/manifest.json",
		base64.StdEncoding.EncodeToString(publicKey),
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := resolver.Resolve(context.Background(), "1.2.3", "linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Version != "1.2.3" || resolved.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("resolved = %#v", resolved)
	}
	if _, err := resolver.Resolve(context.Background(), "1.2.3", "linux", "arm64"); err != nil {
		t.Fatal(err)
	}
	if artifactReads.Load() != 1 {
		t.Fatalf("artifact reads = %d, want 1", artifactReads.Load())
	}
}

func TestResolveRejectsMutableVersion(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewResolver(
		"https://downloads.example/runner/{version}/manifest.json",
		base64.StdEncoding.EncodeToString(publicKey),
		http.DefaultClient,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), "latest", "linux", "amd64"); err == nil {
		t.Fatal("expected mutable version to fail")
	}
}

func TestResolveRejectsManifestVersionMismatch(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"1.2.4","protocol_version":"runner/v1","artifacts":[]}`))
	}))
	defer server.Close()

	resolver, err := NewResolver(
		server.URL+"/{version}",
		base64.StdEncoding.EncodeToString(publicKey),
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), "1.2.3", "linux", "amd64"); err == nil {
		t.Fatal("expected version mismatch")
	}
}
