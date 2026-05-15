package tlsclient_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/tlsclient"
)

// generateSelfSignedCert generates a self-signed cert and returns (certPEM, keyPEM, tlsCert).
func generateSelfSignedCert(t *testing.T) ([]byte, []byte, tls.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		IsCA:         true,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	return certPEM, keyPEM, tlsCert
}

func writeTempFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

func TestNewHTTPClient_NoConfig(t *testing.T) {
	cfg := tlsclient.Config{}
	client, err := tlsclient.NewHTTPClient(cfg, 10*time.Second)
	require.NoError(t, err)
	require.NotNil(t, client)
	tr, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	// No custom TLS config set when nothing is configured.
	assert.Nil(t, tr.TLSClientConfig)
}

func TestNewHTTPClient_InvalidTimeout(t *testing.T) {
	_, err := tlsclient.NewHTTPClient(tlsclient.Config{}, 0)
	require.Error(t, err)
}

func TestNewHTTPClient_InsecureSkipVerify(t *testing.T) {
	cfg := tlsclient.Config{InsecureSkipVerify: true}
	client, err := tlsclient.NewHTTPClient(cfg, 5*time.Second)
	require.NoError(t, err)
	tr := client.Transport.(*http.Transport)
	require.NotNil(t, tr.TLSClientConfig)
	assert.True(t, tr.TLSClientConfig.InsecureSkipVerify)
}

func TestNewHTTPClient_CustomRootCA(t *testing.T) {
	dir := t.TempDir()
	certPEM, _, tlsCert := generateSelfSignedCert(t)
	caPath := writeTempFile(t, dir, "ca.pem", certPEM)

	// Start a TLS server using our self-signed cert.
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{tlsCert}}
	srv.StartTLS()
	defer srv.Close()

	cfg := tlsclient.Config{RootCAFile: caPath}
	client, err := tlsclient.NewHTTPClient(cfg, 5*time.Second)
	require.NoError(t, err)

	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewHTTPClient_MissingRootCAFile(t *testing.T) {
	cfg := tlsclient.Config{RootCAFile: "/nonexistent/ca.pem"}
	_, err := tlsclient.NewHTTPClient(cfg, 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS_ROOT_CA_FILE")
}

func TestNewHTTPClient_ClientCertMissingKey(t *testing.T) {
	dir := t.TempDir()
	certPEM, _, _ := generateSelfSignedCert(t)
	certPath := writeTempFile(t, dir, "cert.pem", certPEM)

	cfg := tlsclient.Config{ClientCertFile: certPath} // no key
	_, err := tlsclient.NewHTTPClient(cfg, 5*time.Second)
	require.Error(t, err)
}

func TestConfigFromEnv_MismatchedCertKey(t *testing.T) {
	dir := t.TempDir()
	certPEM, _, _ := generateSelfSignedCert(t)
	certPath := writeTempFile(t, dir, "cert.pem", certPEM)

	t.Setenv(tlsclient.EnvClientCert, certPath)
	t.Setenv(tlsclient.EnvClientKey, "")

	_, err := tlsclient.ConfigFromEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), tlsclient.EnvClientCert)
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	t.Setenv(tlsclient.EnvRootCA, "")
	t.Setenv(tlsclient.EnvClientCert, "")
	t.Setenv(tlsclient.EnvClientKey, "")
	t.Setenv(tlsclient.EnvInsecure, "")

	cfg, err := tlsclient.ConfigFromEnv()
	require.NoError(t, err)
	assert.Empty(t, cfg.RootCAFile)
	assert.Empty(t, cfg.ClientCertFile)
	assert.Empty(t, cfg.ClientKeyFile)
	assert.False(t, cfg.InsecureSkipVerify)
}
