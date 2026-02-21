package ssh

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const AuthMethodSSHKey = "ssh_key"
const AuthMethodPassword = "password"

// Client connects to an SSH server and runs commands.
// Supports either SSH key or password authentication.
type Client struct {
	Host     string
	Port     int
	Username string

	// For key auth
	PrivateKey []byte
	Passphrase []byte

	// For password auth
	Password []byte

	authMethod string
	conn       *ssh.Client
}

type CommandResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

func NewClientKey(host string, port int, username string, privateKey, passphrase []byte) *Client {
	return &Client{
		Host:       host,
		Port:       port,
		Username:   username,
		PrivateKey: privateKey,
		Passphrase: passphrase,
		authMethod: AuthMethodSSHKey,
	}
}

func NewClientPassword(host string, port int, username string, password []byte) *Client {
	return &Client{
		Host:       host,
		Port:       port,
		Username:   username,
		Password:   password,
		authMethod: AuthMethodPassword,
	}
}

func (c *Client) Connect() (*ssh.Client, error) {
	if c.conn != nil {
		_, _, err := c.conn.SendRequest("keepalive@superplane", false, nil)
		if err == nil {
			return c.conn, nil
		}
		_ = c.conn.Close()
		c.conn = nil
	}

	var auth []ssh.AuthMethod
	switch c.authMethod {
	case AuthMethodSSHKey:
		signer, err := c.getSigner()
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	case AuthMethodPassword:
		auth = []ssh.AuthMethod{ssh.Password(string(c.Password))}
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", c.authMethod)
	}

	config := &ssh.ClientConfig{
		User:            c.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", c.Host, c.Port)
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	c.conn = conn
	return conn, nil
}

func (c *Client) getSigner() (ssh.Signer, error) {
	keyBytes := normalizePrivateKey(c.PrivateKey)

	if len(keyBytes) == 0 {
		return nil, fmt.Errorf("private key is empty after normalization")
	}

	if signer, err := parseSigner(keyBytes, c.Passphrase); err == nil {
		return signer, nil
	}

	if looksLikeBase64(string(keyBytes)) && !looksLikePrivateKeyBlock(string(keyBytes)) {
		if decoded, decErr := decodeBase64(string(keyBytes)); decErr == nil {
			decoded = normalizePrivateKey(decoded)
			signer2, parseErr := parseSigner(decoded, c.Passphrase)
			if parseErr == nil {
				return signer2, nil
			}
			return nil, parseErr
		}
	}

	keyPreview := string(keyBytes)
	if len(keyPreview) > 50 {
		keyPreview = keyPreview[:50] + "..."
	}
	return nil, fmt.Errorf(
		"invalid private key format (expected PEM/OpenSSH block). Key length: %d, starts with: %q",
		len(keyBytes), keyPreview,
	)
}

func parseSigner(keyBytes []byte, passphrase []byte) (ssh.Signer, error) {
	if len(passphrase) > 0 {
		return ssh.ParsePrivateKeyWithPassphrase(keyBytes, passphrase)
	}
	return ssh.ParsePrivateKey(keyBytes)
}

func (c *Client) ExecuteCommand(command string, timeout time.Duration) (*CommandResult, error) {
	conn, err := c.Connect()
	if err != nil {
		return nil, err
	}

	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if timeout > 0 {
		go func() {
			time.Sleep(timeout)
			_ = session.Close()
		}()
	}

	err = session.Run(command)
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			exitCode = exitError.ExitStatus()
		} else {
			return &CommandResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String() + err.Error(),
				ExitCode: -1,
			}, nil
		}
	}

	return &CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

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

	if !strings.Contains(s, "\n") && looksLikePrivateKeyBlock(s) {
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
			if strings.Contains(s, beginMarker) && strings.Contains(s, endMarker) {
				s = strings.Replace(s, beginMarker+" ", beginMarker+"\n", 1)
				s = strings.Replace(s, " "+endMarker, "\n"+endMarker, 1)
				lines := strings.Split(s, "\n")
				for i := range lines {
					if !strings.HasPrefix(lines[i], "-----") {
						lines[i] = strings.ReplaceAll(lines[i], " ", "")
					}
				}
				s = strings.Join(lines, "\n")
				break
			}
		}
	}

	s = strings.TrimSpace(s)
	return []byte(s)
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
