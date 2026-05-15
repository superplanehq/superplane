// Package tlsclient builds outbound HTTP/TLS settings from environment variables.
// It supports extra trusted root CAs, optional client certificates (mTLS),
// and an insecure-skip-verify escape hatch for development.
//
// Environment variables (all optional):
//
//	TLS_ROOT_CA_FILE          – path to PEM file with extra trusted root CA(s)
//	TLS_CLIENT_CERT_FILE      – path to PEM client certificate (mTLS)
//	TLS_CLIENT_KEY_FILE       – path to PEM private key for the client cert (required when cert is set)
//	TLS_INSECURE_SKIP_VERIFY  – "true"/"1"/"yes" disables TLS verification (dev only)
package tlsclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	EnvRootCA     = "TLS_ROOT_CA_FILE"
	EnvClientCert = "TLS_CLIENT_CERT_FILE"
	EnvClientKey  = "TLS_CLIENT_KEY_FILE"
	EnvInsecure   = "TLS_INSECURE_SKIP_VERIFY"
)

// Config holds TLS configuration for outbound HTTP connections.
type Config struct {
	// RootCAFile is the path to a PEM file containing extra trusted root CAs.
	RootCAFile string
	// ClientCertFile and ClientKeyFile are the paths to the client certificate and key (mTLS).
	ClientCertFile string
	ClientKeyFile  string
	// InsecureSkipVerify disables TLS certificate verification. For development only.
	InsecureSkipVerify bool
}

// ConfigFromEnv reads TLS configuration from environment variables.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		RootCAFile:         strings.TrimSpace(os.Getenv(EnvRootCA)),
		ClientCertFile:     strings.TrimSpace(os.Getenv(EnvClientCert)),
		ClientKeyFile:      strings.TrimSpace(os.Getenv(EnvClientKey)),
		InsecureSkipVerify: envTruthy(EnvInsecure),
	}

	if cfg.ClientCertFile != "" && cfg.ClientKeyFile == "" {
		return Config{}, fmt.Errorf("%s: set both %s and %s for client TLS, or unset both",
			EnvClientCert, EnvClientCert, EnvClientKey)
	}
	if cfg.ClientCertFile == "" && cfg.ClientKeyFile != "" {
		return Config{}, fmt.Errorf("%s: set both %s and %s for client TLS, or unset both",
			EnvClientKey, EnvClientCert, EnvClientKey)
	}

	return cfg, nil
}

// NewHTTPClient returns an *http.Client with a TLS transport built from cfg and the given timeout.
// When cfg has no customizations, the returned client uses standard library TLS defaults.
func NewHTTPClient(cfg Config, timeout time.Duration) (*http.Client, error) {
	if timeout <= 0 {
		return nil, errors.New("tlsclient: timeout must be positive")
	}
	tr, err := newTransport(cfg)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: tr, Timeout: timeout}, nil
}

// NewHTTPClientFromEnv is a convenience wrapper that reads config from environment variables.
func NewHTTPClientFromEnv(timeout time.Duration) (*http.Client, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewHTTPClient(cfg, timeout)
}

func newTransport(cfg Config) (*http.Transport, error) {
	tlsCfg, err := buildTLSConfig(cfg)
	if err != nil {
		return nil, err
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = tlsCfg
	return tr, nil
}

func buildTLSConfig(cfg Config) (*tls.Config, error) {
	// No customization needed — let the stdlib use its defaults.
	if cfg.RootCAFile == "" && cfg.ClientCertFile == "" && !cfg.InsecureSkipVerify {
		return nil, nil
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // intentional dev escape-hatch
	}

	if cfg.RootCAFile != "" {
		pemData, err := os.ReadFile(cfg.RootCAFile)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", EnvRootCA, err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pemData) {
			return nil, fmt.Errorf("%s: no valid PEM certificates found", EnvRootCA)
		}
		tlsCfg.RootCAs = pool
	}

	if cfg.ClientCertFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("%s / %s: %w", EnvClientCert, EnvClientKey, err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}

func envTruthy(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}
