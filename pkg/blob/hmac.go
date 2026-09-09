package blob

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SignedURLMarkerParam = "sp_file"
	SignedURLMarkerValue = "1"
)

func SigningKey() ([]byte, error) {
	for _, name := range []string{"BLOB_STORAGE_SIGNING_KEY", "ENCRYPTION_KEY", "JWT_SECRET"} {
		value := strings.TrimSpace(os.Getenv(name))
		if value != "" {
			return []byte(value), nil
		}
	}
	return nil, fmt.Errorf("BLOB_STORAGE_SIGNING_KEY is not set")
}

func PublicBaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("BASE_URL")), "/")
	if base != "" {
		return base
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

func FileAccessURL(fileID uuid.UUID, ttl time.Duration, key []byte) (string, error) {
	if ttl <= 0 {
		ttl = time.Hour
	}
	expires := time.Now().Add(ttl).Unix()
	sig, err := SignFileAccess(fileID, expires, key)
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("expires", strconv.FormatInt(expires, 10))
	query.Set("sig", sig)
	query.Set(SignedURLMarkerParam, SignedURLMarkerValue)
	return fmt.Sprintf("%s/api/v1/public/files/%s?%s", PublicBaseURL(), fileID, query.Encode()), nil
}

func SignFileAccess(fileID uuid.UUID, expires int64, key []byte) (string, error) {
	if fileID == uuid.Nil {
		return "", fmt.Errorf("file id is required")
	}
	if len(key) == 0 {
		return "", fmt.Errorf("signing key is required")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = fmt.Fprintf(mac, "%s.%d", fileID, expires)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func VerifyFileAccess(fileID uuid.UUID, expires int64, sig string, key []byte) error {
	if time.Now().Unix() > expires {
		return fmt.Errorf("file URL has expired")
	}
	expected, err := SignFileAccess(fileID, expires, key)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(strings.ToLower(expected))) {
		return fmt.Errorf("invalid file URL signature")
	}
	return nil
}

var httpURLPattern = regexp.MustCompile(`https?://[^\s)\]>"']+`)

func SignedFileURLs(text string) []string {
	seen := map[string]struct{}{}
	var urls []string
	for _, match := range httpURLPattern.FindAllString(text, -1) {
		parsed, err := url.Parse(match)
		if err != nil {
			continue
		}
		if parsed.Query().Get(SignedURLMarkerParam) != SignedURLMarkerValue {
			continue
		}
		if _, exists := seen[match]; exists {
			continue
		}
		seen[match] = struct{}{}
		urls = append(urls, match)
	}
	return urls
}

func ResolveDownloadURL(ctx context.Context, provider Provider, storageKey string, fileID uuid.UUID, ttl time.Duration) (string, error) {
	if provider == nil {
		return "", ErrProviderNotConfigured
	}
	signed, err := provider.SignedGetURL(ctx, storageKey, ttl)
	if err == nil {
		return signed, nil
	}
	if !errors.Is(err, ErrSignedURLUnsupported) {
		return "", err
	}
	key, err := SigningKey()
	if err != nil {
		return "", err
	}
	return FileAccessURL(fileID, ttl, key)
}
