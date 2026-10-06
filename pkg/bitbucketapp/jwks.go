package bitbucketapp

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v4"
)

const jwksTTL = time.Hour

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type keyCache struct {
	mu        sync.Mutex
	url       string
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

var (
	jwksMu     sync.Mutex
	jwksCaches = map[string]*keyCache{}
	jwksClient = &http.Client{Timeout: 10 * time.Second}
)

// VerificationKeyfunc verifies Forge Invocation Tokens with the published JWKS.
// Keys are cached. A missing key id refreshes the cache once.
func VerificationKeyfunc(ctx context.Context, jwksURL string) (jwtlib.Keyfunc, error) {
	if jwksURL == "" {
		return nil, fmt.Errorf("forge jwks url is required")
	}
	cache := cacheFor(jwksURL)
	return func(token *jwtlib.Token) (any, error) {
		if token.Method == nil || token.Method.Alg() != jwtlib.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("forge invocation token must use RS256")
		}
		kid, _ := token.Header["kid"].(string)
		return cache.key(ctx, kid)
	}, nil
}

func cacheFor(jwksURL string) *keyCache {
	jwksMu.Lock()
	defer jwksMu.Unlock()
	cache, ok := jwksCaches[jwksURL]
	if ok {
		return cache
	}
	cache = &keyCache{url: jwksURL, keys: map[string]*rsa.PublicKey{}}
	jwksCaches[jwksURL] = cache
	return cache
}

func (c *keyCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.refreshIfNeeded(ctx, kid); err != nil && len(c.keys) == 0 {
		return nil, err
	}
	if kid != "" {
		key := c.keys[kid]
		if key == nil {
			return nil, fmt.Errorf("forge verification key was not found")
		}
		return key, nil
	}
	if len(c.keys) == 1 {
		for _, key := range c.keys {
			return key, nil
		}
	}
	return nil, fmt.Errorf("forge invocation token has no key id")
}

func (c *keyCache) refreshIfNeeded(ctx context.Context, kid string) error {
	stale := c.fetchedAt.IsZero() || time.Since(c.fetchedAt) > jwksTTL
	missing := kid != "" && c.keys[kid] == nil
	if !stale && !missing {
		return nil
	}
	return c.refresh(ctx)
}

func (c *keyCache) refresh(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	response, err := jwksClient.Do(request)
	if err != nil {
		return fmt.Errorf("load forge verification keys: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("load forge verification keys: status %d", response.StatusCode)
	}
	var document jwksDocument
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		return fmt.Errorf("decode forge verification keys: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, entry := range document.Keys {
		if entry.Kty != "" && entry.Kty != "RSA" {
			continue
		}
		if entry.N == "" || entry.E == "" {
			continue
		}
		publicKey, err := RSAPublicKey(entry.N, entry.E)
		if err != nil {
			return err
		}
		keys[entry.Kid] = publicKey
	}
	if len(keys) == 0 {
		return fmt.Errorf("forge verification keys are empty")
	}
	c.keys = keys
	c.fetchedAt = time.Now()
	return nil
}
