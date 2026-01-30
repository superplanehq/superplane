package ssh

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	Host       string
	Port       int
	Username   string
	PrivateKey []byte
	Passphrase []byte
	conn       *ssh.Client
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type HostMetadataResult struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Kernel       string `json:"kernel"`
	Architecture string `json:"architecture"`
	Uptime       string `json:"uptime"`
	DiskUsage    string `json:"diskUsage"`
	MemoryInfo   string `json:"memoryInfo"`
}

// ✅ New: build client from the already-decoded integration Configuration
func NewClientFromConfig(cfg Configuration) (*Client, error) {
	host := strings.TrimSpace(cfg.Host)
	username := strings.TrimSpace(cfg.Username)

	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if strings.TrimSpace(cfg.PrivateKey) == "" {
		return nil, fmt.Errorf("privateKey is required")
	}

	port := cfg.Port
	if port == 0 {
		port = 22
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}

	return &Client{
		Host:       host,
		Port:       port,
		Username:   username,
		PrivateKey: []byte(cfg.PrivateKey),
		Passphrase: []byte(cfg.Passphrase),
	}, nil
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

	signer, err := c.getSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User:            c.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", c.Host, c.Port)
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	c.conn = conn
	return conn, nil
}

func (c *Client) getSigner() (ssh.Signer, error) {
	keyBytes := normalizePrivateKey(c.PrivateKey)

	// Check if key is empty after normalization
	if len(keyBytes) == 0 {
		return nil, fmt.Errorf("private key is empty after normalization (original length: %d)", len(c.PrivateKey))
	}

	// 1) Try parsing as-is
	if signer, err := parseSigner(keyBytes, c.Passphrase); err == nil {
		return signer, nil
	} else {
		// If it clearly doesn't look like a PEM/OpenSSH key, try base64 decode fallback
		if looksLikeBase64(string(keyBytes)) && !looksLikePrivateKeyBlock(string(keyBytes)) {
			if decoded, decErr := decodeBase64(string(keyBytes)); decErr == nil {
				decoded = normalizePrivateKey(decoded)
				if signer2, err2 := parseSigner(decoded, c.Passphrase); err2 == nil {
					return signer2, nil
				} else {
					return nil, err2
				}
			}
		}

		// 2) Give a much more actionable error with diagnostic info
		keyPreview := string(keyBytes)
		if len(keyPreview) > 50 {
			keyPreview = keyPreview[:50] + "..."
		}

		return nil, fmt.Errorf(
			"invalid private key format (expected PEM/OpenSSH private key block like '-----BEGIN ... PRIVATE KEY-----'). "+
				"If you pasted a public key (starts with 'ssh-'), generate/export the PRIVATE key instead. "+
				"Key length: %d bytes, starts with: %q. "+
				"Original error: %v",
			len(keyBytes), keyPreview, err,
		)
	}
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
		return nil, fmt.Errorf("failed to create session: %v", err)
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

func (c *Client) ExecuteScript(script string, interpreter string, timeout time.Duration) (*CommandResult, error) {
	if interpreter == "" {
		interpreter = "bash"
	}

	conn, err := c.Connect()
	if err != nil {
		return nil, err
	}

	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Start(interpreter); err != nil {
		return nil, fmt.Errorf("failed to start interpreter: %v", err)
	}

	if _, err := stdin.Write([]byte(script)); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("failed to write script: %v", err)
	}
	_ = stdin.Close()

	done := make(chan error, 1)
	go func() { done <- session.Wait() }()

	var waitErr error
	if timeout > 0 {
		select {
		case waitErr = <-done:
		case <-time.After(timeout):
			_ = session.Close()
			return &CommandResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String() + "command timed out",
				ExitCode: -1,
			}, nil
		}
	} else {
		waitErr = <-done
	}

	if waitErr != nil {
		if exitError, ok := waitErr.(*ssh.ExitError); ok {
			return &CommandResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: exitError.ExitStatus(),
			}, nil
		}
		return &CommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String() + waitErr.Error(),
			ExitCode: -1,
		}, nil
	}

	return &CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}, nil
}

func (c *Client) GetHostMetadata() (*HostMetadataResult, error) {
	conn, err := c.Connect()
	if err != nil {
		return nil, err
	}

	result := &HostMetadataResult{}

	if hostname, err := c.executeSimpleCommand(conn, "hostname"); err == nil {
		result.Hostname = hostname
	}
	if osInfo, err := c.executeSimpleCommand(conn, "uname -a"); err == nil {
		result.OS = osInfo
		if kernel, err := c.executeSimpleCommand(conn, "uname -r"); err == nil {
			result.Kernel = kernel
		}
		if arch, err := c.executeSimpleCommand(conn, "uname -m"); err == nil {
			result.Architecture = arch
		}
	}
	if uptime, err := c.executeSimpleCommand(conn, "uptime"); err == nil {
		result.Uptime = uptime
	}
	if diskUsage, err := c.executeSimpleCommand(conn, "df -h"); err == nil {
		result.DiskUsage = diskUsage
	}
	if memoryInfo, err := c.executeSimpleCommand(conn, "free -m 2>/dev/null || vm_stat 2>/dev/null || sysctl hw.memsize 2>/dev/null || echo 'Memory info not available'"); err == nil {
		result.MemoryInfo = memoryInfo
	}

	return result, nil
}

func (c *Client) executeSimpleCommand(conn *ssh.Client, command string) (string, error) {
	session, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.Output(command)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(output)), nil
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

	// Remove surrounding quotes (common when stored as JSON string)
	s = strings.Trim(s, `"`)
	s = strings.Trim(s, `'`)

	// Normalize Windows newlines
	s = strings.ReplaceAll(s, "\r\n", "\n")

	// Case 1: literal "\n" sequences (user pasted escaped newlines)
	// Handle even if there are some real newlines mixed in
	if strings.Contains(s, `\n`) {
		s = strings.ReplaceAll(s, `\n`, "\n")
	}

	// Remove accidental leading/trailing spaces on each line
	if strings.Contains(s, "\n") {
		lines := strings.Split(s, "\n")
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " \t")
		}
		s = strings.Join(lines, "\n")
	}

	// Case 2: flattened key in one line with spaces (supports OpenSSH, RSA, EC, DSA keys)
	if !strings.Contains(s, "\n") && looksLikePrivateKeyBlock(s) {
		// List of common private key types
		keyTypes := []string{
			"OPENSSH PRIVATE KEY",
			"RSA PRIVATE KEY",
			"EC PRIVATE KEY",
			"DSA PRIVATE KEY",
			"PRIVATE KEY", // Generic PKCS#8
		}

		for _, keyType := range keyTypes {
			beginMarker := "-----BEGIN " + keyType + "-----"
			endMarker := "-----END " + keyType + "-----"

			if strings.Contains(s, beginMarker) && strings.Contains(s, endMarker) {
				s = strings.Replace(s, beginMarker+" ", beginMarker+"\n", 1)
				s = strings.Replace(s, " "+endMarker, "\n"+endMarker, 1)

				// Remove spaces in the base64 body line (if any)
				lines := strings.Split(s, "\n")
				for i := range lines {
					lines[i] = strings.ReplaceAll(lines[i], " ", "")
				}
				s = strings.Join(lines, "\n")
				break
			}
		}
	}

	// Ensure trailing newline doesn't hurt, but keep clean
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
	// Fast reject if it contains PEM markers or obvious non-base64 chars
	if strings.Contains(s, "BEGIN") || strings.Contains(s, "END") {
		return false
	}
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '+' || r == '/' || r == '=' ||
			r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			continue
		}
		return false
	}
	return true
}

func decodeBase64(s string) ([]byte, error) {
	// Remove whitespace/newlines
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\t", "")

	// Standard base64 decode first, then URL-safe as fallback
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}
