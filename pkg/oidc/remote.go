package oidc

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type jwksDocument struct {
	Keys []PublicJWK `json:"keys"`
}

func ValidateRemote(client *http.Client, token, baseURL string) (map[string]any, error) {
	if client == nil {
		client = http.DefaultClient
	}

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	discovery, err := fetchDiscoveryDocument(client, baseURL+"/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}

	jwksURL := strings.TrimSpace(discovery.JWKSURI)
	if jwksURL == "" {
		jwksURL = baseURL + "/.well-known/jwks.json"
	}

	publicKeys, err := fetchPublicKeys(client, jwksURL)
	if err != nil {
		return nil, err
	}

	return ValidateToken(token, discovery.Issuer, publicKeys)
}

func fetchDiscoveryDocument(client *http.Client, discoveryURL string) (discoveryDocument, error) {
	response, err := client.Get(discoveryURL)
	if err != nil {
		return discoveryDocument{}, fmt.Errorf("fetch OIDC discovery document: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return discoveryDocument{}, fmt.Errorf("fetch OIDC discovery document: unexpected status %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return discoveryDocument{}, fmt.Errorf("read OIDC discovery document: %w", err)
	}

	var document discoveryDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return discoveryDocument{}, fmt.Errorf("parse OIDC discovery document: %w", err)
	}

	if strings.TrimSpace(document.Issuer) == "" {
		return discoveryDocument{}, fmt.Errorf("OIDC discovery document is missing issuer")
	}

	return document, nil
}

func fetchPublicKeys(client *http.Client, jwksURL string) (map[string]*rsa.PublicKey, error) {
	response, err := client.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch JWKS: unexpected status %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read JWKS: %w", err)
	}

	var document jwksDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("parse JWKS: %w", err)
	}

	return PublicKeysFromJWKs(document.Keys)
}
