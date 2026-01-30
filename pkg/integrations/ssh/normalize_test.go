package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test normalizePrivateKey with various user input formats
func Test__NormalizePrivateKey__UserInputFormats(t *testing.T) {
	// Sample key structure for testing (not a real key)
	validKeyWithNewlines := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBHK9tolyHskjIGBp8V78J3F11jf7wQrEb1jW04E6G/vwAAAJAh4h3CIeId
wgAAAAtzc2gtZWQyNTUxOQAAACBHK9tolyHskjIGBp8V78J3F11jf7wQrEb1jW04E6G/vw
-----END OPENSSH PRIVATE KEY-----`

	t.Run("key with literal backslash-n sequences", func(t *testing.T) {
		// User pastes: -----BEGIN OPENSSH PRIVATE KEY-----\nb3Blbn...\n-----END OPENSSH PRIVATE KEY-----
		inputWithEscapedNewlines := `-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW\nQyNTUxOQAAACBHK9tolyHskjIGBp8V78J3F11jf7wQrEb1jW04E6G/vwAAAJAh4h3CIeId\nwgAAAAtzc2gtZWQyNTUxOQAAACBHK9tolyHskjIGBp8V78J3F11jf7wQrEb1jW04E6G/vw\n-----END OPENSSH PRIVATE KEY-----`

		result := normalizePrivateKey([]byte(inputWithEscapedNewlines))

		// Should convert \n to actual newlines
		assert.Contains(t, string(result), "\n")
		assert.Contains(t, string(result), "-----BEGIN OPENSSH PRIVATE KEY-----")
		assert.Contains(t, string(result), "-----END OPENSSH PRIVATE KEY-----")
	})

	t.Run("key with Windows CRLF newlines", func(t *testing.T) {
		inputWithCRLF := "-----BEGIN OPENSSH PRIVATE KEY-----\r\nb3BlbnNzaC1rZXktdjE\r\n-----END OPENSSH PRIVATE KEY-----"

		result := normalizePrivateKey([]byte(inputWithCRLF))

		// Should normalize to Unix newlines
		assert.NotContains(t, string(result), "\r\n")
		assert.Contains(t, string(result), "\n")
	})

	t.Run("key wrapped in double quotes", func(t *testing.T) {
		inputWithQuotes := `"-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjE
-----END OPENSSH PRIVATE KEY-----"`

		result := normalizePrivateKey([]byte(inputWithQuotes))

		// Should strip quotes
		assert.NotContains(t, string(result), `"`)
		assert.True(t, len(result) > 0)
	})

	t.Run("key wrapped in single quotes", func(t *testing.T) {
		inputWithQuotes := `'-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjE
-----END OPENSSH PRIVATE KEY-----'`

		result := normalizePrivateKey([]byte(inputWithQuotes))

		// Should strip quotes
		assert.NotContains(t, string(result), `'`)
	})

	t.Run("key with trailing whitespace on lines", func(t *testing.T) {
		inputWithTrailingSpaces := "-----BEGIN OPENSSH PRIVATE KEY-----  \nb3BlbnNzaC1rZXktdjE   \n-----END OPENSSH PRIVATE KEY-----\t"

		result := normalizePrivateKey([]byte(inputWithTrailingSpaces))

		// Should trim trailing whitespace from lines
		lines := string(result)
		assert.NotContains(t, lines, "-----  \n")
		assert.NotContains(t, lines, "   \n")
	})

	t.Run("RSA key pasted as single line with spaces", func(t *testing.T) {
		inputSingleLine := "-----BEGIN RSA PRIVATE KEY----- MIIEpQIBAAKCAQ -----END RSA PRIVATE KEY-----"

		result := normalizePrivateKey([]byte(inputSingleLine))

		// Should reconstruct proper format with newlines
		assert.Contains(t, string(result), "-----BEGIN RSA PRIVATE KEY-----")
		assert.Contains(t, string(result), "\n")
		assert.Contains(t, string(result), "-----END RSA PRIVATE KEY-----")
	})

	t.Run("EC key pasted as single line with spaces", func(t *testing.T) {
		inputSingleLine := "-----BEGIN EC PRIVATE KEY----- MHQCAQEEIFBLaw -----END EC PRIVATE KEY-----"

		result := normalizePrivateKey([]byte(inputSingleLine))

		// Should reconstruct proper format with newlines
		assert.Contains(t, string(result), "-----BEGIN EC PRIVATE KEY-----")
		assert.Contains(t, string(result), "\n")
		assert.Contains(t, string(result), "-----END EC PRIVATE KEY-----")
	})

	t.Run("mixed escaped and real newlines", func(t *testing.T) {
		// Key has some real newlines and some escaped \n
		inputMixed := "-----BEGIN OPENSSH PRIVATE KEY-----\nb3Blbn\\nNzaC1r\\n-----END OPENSSH PRIVATE KEY-----"

		result := normalizePrivateKey([]byte(inputMixed))

		// Should convert all \n to real newlines
		assert.NotContains(t, string(result), `\n`) // No escaped \n remaining
		assert.Contains(t, string(result), "\n")    // Has real newlines
	})
}

// Test parseHostIdentifier with various port formats
func Test__ParseHostIdentifier__PortFormats(t *testing.T) {
	t.Run("standard format with port", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("user@example.com:22")

		require.NoError(t, err)
		assert.Equal(t, "user", username)
		assert.Equal(t, "example.com", host)
		assert.Equal(t, 22, port)
	})

	t.Run("custom port", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("admin@server.local:2222")

		require.NoError(t, err)
		assert.Equal(t, "admin", username)
		assert.Equal(t, "server.local", host)
		assert.Equal(t, 2222, port)
	})

	t.Run("no port defaults to 22", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("root@192.168.1.100")

		require.NoError(t, err)
		assert.Equal(t, "root", username)
		assert.Equal(t, "192.168.1.100", host)
		assert.Equal(t, 22, port) // Default port
	})

	t.Run("IPv6 with brackets and port", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("user@[::1]:2222")

		require.NoError(t, err)
		assert.Equal(t, "user", username)
		assert.Equal(t, "::1", host)
		assert.Equal(t, 2222, port)
	})

	t.Run("IPv6 without port defaults to 22", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("user@[::1]")

		require.NoError(t, err)
		assert.Equal(t, "user", username)
		assert.Equal(t, "::1", host)
		assert.Equal(t, 22, port) // Default port
	})

	t.Run("IPv6 localhost with brackets", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("user@[2001:db8::1]:22")

		require.NoError(t, err)
		assert.Equal(t, "user", username)
		assert.Equal(t, "2001:db8::1", host)
		assert.Equal(t, 22, port)
	})

	t.Run("IPv6 full address without port", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("admin@[fe80::1%eth0]")

		require.NoError(t, err)
		assert.Equal(t, "admin", username)
		assert.Equal(t, "fe80::1%eth0", host)
		assert.Equal(t, 22, port)
	})

	t.Run("IPv6 missing closing bracket fails", func(t *testing.T) {
		_, _, _, err := parseHostIdentifier("user@[::1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing closing bracket")
	})

	t.Run("IPv6 with trailing colon but no port fails", func(t *testing.T) {
		_, _, _, err := parseHostIdentifier("user@[::1]:")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty port")
	})

	t.Run("whitespace is trimmed", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("  user@example.com:22  ")

		require.NoError(t, err)
		assert.Equal(t, "user", username)
		assert.Equal(t, "example.com", host)
		assert.Equal(t, 22, port)
	})

	t.Run("empty string fails", func(t *testing.T) {
		_, _, _, err := parseHostIdentifier("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty host value")
	})

	t.Run("missing username fails", func(t *testing.T) {
		_, _, _, err := parseHostIdentifier("example.com:22")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing username")
	})

	t.Run("empty username fails", func(t *testing.T) {
		_, _, _, err := parseHostIdentifier("@example.com:22")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing username")
	})

	t.Run("host with subdomain and port", func(t *testing.T) {
		username, host, port, err := parseHostIdentifier("deploy@prod.api.example.com:2022")

		require.NoError(t, err)
		assert.Equal(t, "deploy", username)
		assert.Equal(t, "prod.api.example.com", host)
		assert.Equal(t, 2022, port)
	})

	t.Run("port out of valid range in NewClientFromConfig", func(t *testing.T) {
		// Port validation happens in NewClientFromConfig
		_, err := NewClientFromConfig(Configuration{
			Host:       "example.com",
			Port:       70000, // Invalid port
			Username:   "user",
			PrivateKey: "fake-key",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid port")
	})

	t.Run("port zero defaults to 22 in NewClientFromConfig", func(t *testing.T) {
		client, err := NewClientFromConfig(Configuration{
			Host:       "example.com",
			Port:       0,
			Username:   "user",
			PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nfake\n-----END OPENSSH PRIVATE KEY-----",
		})

		require.NoError(t, err)
		assert.Equal(t, 22, client.Port)
	})
}

// Test looksLikeBase64 function
func Test__LooksLikeBase64(t *testing.T) {
	t.Run("valid base64 string", func(t *testing.T) {
		assert.True(t, looksLikeBase64("SGVsbG8gV29ybGQ="))
	})

	t.Run("base64 with newlines", func(t *testing.T) {
		assert.True(t, looksLikeBase64("SGVsbG8g\nV29ybGQ="))
	})

	t.Run("empty string returns false", func(t *testing.T) {
		assert.False(t, looksLikeBase64(""))
	})

	t.Run("PEM markers return false", func(t *testing.T) {
		assert.False(t, looksLikeBase64("-----BEGIN OPENSSH PRIVATE KEY-----"))
	})

	t.Run("string with invalid characters returns false", func(t *testing.T) {
		assert.False(t, looksLikeBase64("Hello World!@#$"))
	})
}

// Test looksLikePrivateKeyBlock function
func Test__LooksLikePrivateKeyBlock(t *testing.T) {
	t.Run("OpenSSH private key", func(t *testing.T) {
		assert.True(t, looksLikePrivateKeyBlock("-----BEGIN OPENSSH PRIVATE KEY-----"))
	})

	t.Run("RSA private key", func(t *testing.T) {
		assert.True(t, looksLikePrivateKeyBlock("-----BEGIN RSA PRIVATE KEY-----"))
	})

	t.Run("EC private key", func(t *testing.T) {
		assert.True(t, looksLikePrivateKeyBlock("-----BEGIN EC PRIVATE KEY-----"))
	})

	t.Run("public key returns false", func(t *testing.T) {
		assert.False(t, looksLikePrivateKeyBlock("ssh-rsa AAAA..."))
	})

	t.Run("random text returns false", func(t *testing.T) {
		assert.False(t, looksLikePrivateKeyBlock("some random text"))
	})
}
