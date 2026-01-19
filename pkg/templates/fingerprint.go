package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
)

// templateFingerprint calculates a fingerprint of the template directory.
// It reads all the files in the directory and calculates a SHA256 hash of the files.
// It returns the hash as a hex string.

func templateDirFingerprint(dir fs.FS) (string, error) {
	entries, err := fs.ReadDir(dir, ".")
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(dir, entry.Name())
		if err != nil {
			return "", err
		}
		_, _ = hasher.Write(data)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
