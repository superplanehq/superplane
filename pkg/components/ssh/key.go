package ssh

import (
	"encoding/base64"
	"strings"
)

const AuthMethodSSHKey = "ssh_key"
const AuthMethodPassword = "password"

// normalizePrivateKey cleans up a stored SSH private key so the `ssh` CLI on the
// runner can parse it. Secrets are frequently pasted with surrounding quotes,
// escaped "\n" sequences (JSON/YAML round-trips), CRLF endings, or as a single
// base64 blob. We repair those forms into a canonical PEM/OpenSSH block written
// verbatim to the key file on the runner.
func normalizePrivateKey(raw []byte) []byte {
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, `"`)
	s = strings.Trim(s, `'`)
	s = strings.ReplaceAll(s, "\r\n", "\n")

	if strings.Contains(s, `\n`) {
		s = strings.ReplaceAll(s, `\n`, "\n")
	}

	if strings.Contains(s, "\n") {
		lines := strings.Split(s, "\n")
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " \t")
		}
		s = strings.Join(lines, "\n")
	}

	// A key stored as a single base64 blob (no PEM markers) is decoded so the
	// runner receives the underlying key material.
	if looksLikeBase64(s) && !looksLikePrivateKeyBlock(s) {
		if decoded, err := decodeBase64(s); err == nil {
			return normalizePrivateKey(decoded)
		}
	}

	if !strings.Contains(s, "\n") && looksLikePrivateKeyBlock(s) {
		s = reflowSingleLinePrivateKey(s)
	}

	return []byte(strings.TrimSpace(s) + "\n")
}

// reflowSingleLinePrivateKey restores newlines in a PEM block that was collapsed
// onto a single line (spaces separating the header, body, and footer).
func reflowSingleLinePrivateKey(s string) string {
	keyTypes := []string{
		"OPENSSH PRIVATE KEY",
		"RSA PRIVATE KEY",
		"EC PRIVATE KEY",
		"DSA PRIVATE KEY",
		"PRIVATE KEY",
	}
	for _, keyType := range keyTypes {
		beginMarker := "-----BEGIN " + keyType + "-----"
		endMarker := "-----END " + keyType + "-----"
		if !strings.Contains(s, beginMarker) || !strings.Contains(s, endMarker) {
			continue
		}
		s = strings.Replace(s, beginMarker+" ", beginMarker+"\n", 1)
		s = strings.Replace(s, " "+endMarker, "\n"+endMarker, 1)
		lines := strings.Split(s, "\n")
		for i := range lines {
			if !strings.HasPrefix(lines[i], "-----") {
				lines[i] = strings.ReplaceAll(lines[i], " ", "")
			}
		}
		return strings.Join(lines, "\n")
	}
	return s
}

func looksLikePrivateKeyBlock(s string) bool {
	return strings.Contains(s, "-----BEGIN ") && strings.Contains(s, " PRIVATE KEY-----")
}

func looksLikeBase64(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.Contains(s, "BEGIN") || strings.Contains(s, "END") {
		return false
	}
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			r == '+' || r == '/' || r == '=' || r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			continue
		}
		return false
	}
	return true
}

func decodeBase64(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\t", "")
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}
