package artifact

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestResolveReadsVersionedReleaseChecksum(t *testing.T) {
	var checksumReads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/runner/v1.2.3/checksums.txt":
			checksumReads.Add(1)
			_, _ = w.Write([]byte(
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  runner-linux-amd64.tar.gz\n" +
					"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  runner-linux-arm64.tar.gz\n",
			))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resolver, err := NewResolver(server.URL+"/runner/", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := resolver.Resolve(context.Background(), "v1.2.3", "linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Version != "v1.2.3" ||
		resolved.URL != server.URL+"/runner/v1.2.3/runner-linux-arm64.tar.gz" ||
		resolved.SHA256 != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("resolved = %#v", resolved)
	}
	if _, err := resolver.Resolve(context.Background(), "v1.2.3", "linux", "arm64"); err != nil {
		t.Fatal(err)
	}
	if checksumReads.Load() != 1 {
		t.Fatalf("checksum reads = %d, want 1", checksumReads.Load())
	}
}

func TestResolveRejectsMutableVersion(t *testing.T) {
	resolver, err := NewResolver("https://downloads.example/runner", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{
		"latest",
		"1.2.3",
		strings.Repeat("a", 40),
		"sha:" + strings.Repeat("a", 39),
		"sha:" + strings.Repeat("A", 40),
	} {
		t.Run(version, func(t *testing.T) {
			if _, err := resolver.Resolve(
				context.Background(),
				version,
				"linux",
				"amd64",
			); err == nil {
				t.Fatalf("expected runner release %q to fail", version)
			}
		})
	}
}

func TestResolveReadsGitSHAReleaseChecksum(t *testing.T) {
	version := "sha:" + strings.Repeat("a", 40)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/runner/"+version+"/checksums.txt" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  runner-linux-amd64.tar.gz\n",
		))
	}))
	defer server.Close()

	resolver, err := NewResolver(server.URL+"/runner", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := resolver.Resolve(context.Background(), version, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Version != version ||
		resolved.URL != server.URL+"/runner/"+version+"/runner-linux-amd64.tar.gz" {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveRejectsMissingArtifactChecksum(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  runner-linux-amd64.tar.gz\n",
		))
	}))
	defer server.Close()

	resolver, err := NewResolver(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), "v1.2.3", "linux", "arm64"); err == nil {
		t.Fatal("expected missing checksum")
	}
}

func TestNewResolverRejectsMutableReleaseURL(t *testing.T) {
	if _, err := NewResolver(
		"https://downloads.example/runner/latest",
		http.DefaultClient,
	); err == nil {
		t.Fatal("expected mutable release URL to fail")
	}
}
