package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMediaToolchainIsWiredInRunnerBake(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	files := map[string][]string{
		"Makefile": {
			"ffmpeg", "ffprobe", "whisper-cli", "ggml-tiny.bin",
		},
		"runner/runner/Dockerfile.local": {
			"ffmpeg", "ffprobe", "whisper-cli", "whisper-cli --help", "install-media-tools.sh", "ggml-tiny.bin",
		},
		"runner/packer/scripts/install.sh": {
			"ffmpeg", "ffprobe", "WHISPER_MODEL=/usr/local/share/whisper/ggml-tiny.bin",
		},
		"runner/packer/scripts/install-media-tools.sh": {
			"ffmpeg", "ffprobe", "whisper-cli", "ggml-tiny.bin", "v1.9.2",
			"GGML_CPU_ARM_ARCH=armv8.2-a+dotprod+fp16",
		},
		"runner/packer/runner.pkr.hcl": {
			"install-media-tools.sh",
		},
	}
	for rel, needles := range files {
		body, err := os.ReadFile(filepath.Join(root, rel))
		require.NoError(t, err, rel)
		for _, needle := range needles {
			assert.Contains(t, string(body), needle, rel)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "runner", "packer", "scripts", "install.sh")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repository root")
	return ""
}
