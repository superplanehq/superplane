package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// TestExecuteScriptDoesNotLeakTimeoutGoroutine guards against a regression
// where the timeout enforcer was a fire-and-forget goroutine that slept for
// the whole timeout even when the command finished first. Each fast command
// with a long timeout stranded one goroutine (and its session) until the
// timeout elapsed. The timer-based enforcer stops immediately on early return.
func TestExecuteScriptDoesNotLeakTimeoutGoroutine(t *testing.T) {
	const (
		username = "tester"
		password = "secret"
		// Long enough that a leaked sleep goroutine would still be parked
		// when the test measures, but short enough to keep the test bounded.
		commandTimeout = 60 * time.Second
		iterations     = 40
	)

	server, err := startEchoSSHServer(username, password)
	require.NoError(t, err)
	defer server.Close()

	client := NewClientPassword("127.0.0.1", server.Port(), username, []byte(password))
	defer client.Close()

	// Warm up the shared connection so its goroutines are part of the
	// baseline and are not mistaken for a leak.
	_, err = client.ExecuteCommand("true", commandTimeout)
	require.NoError(t, err)

	baseline := settledGoroutineCount()

	for i := 0; i < iterations; i++ {
		result, err := client.ExecuteCommand("true", commandTimeout)
		require.NoError(t, err)
		require.Equal(t, 0, result.ExitCode)
	}

	leaked := settledGoroutineCount() - baseline
	require.Lessf(
		t,
		leaked,
		iterations/2,
		"expected timeout goroutines to be released after each command; %d of %d still running",
		leaked,
		iterations,
	)
}

// settledGoroutineCount lets short-lived goroutines finish, then samples the
// live goroutine count. It polls so transient session teardown does not skew
// the reading.
func settledGoroutineCount() int {
	previous := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		runtime.Gosched()
		current := runtime.NumGoroutine()
		if current >= previous {
			return current
		}
		previous = current
	}
	return previous
}

type echoSSHServer struct {
	listener net.Listener
	config   *ssh.ServerConfig
}

// startEchoSSHServer stands up an in-process SSH server that accepts any exec
// request and immediately replies with exit status 0.
func startEchoSSHServer(username, password string) (*echoSSHServer, error) {
	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.NewSignerFromKey(hostKey)
	if err != nil {
		return nil, err
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(metadata ssh.ConnMetadata, candidate []byte) (*ssh.Permissions, error) {
			if metadata.User() == username && string(candidate) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid SSH credentials for %s", metadata.User())
		},
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	server := &echoSSHServer{listener: listener, config: config}
	go server.accept()
	return server, nil
}

func (s *echoSSHServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *echoSSHServer) Close() {
	_ = s.listener.Close()
}

func (s *echoSSHServer) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *echoSSHServer) handleConnection(conn net.Conn) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(requests)

	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(ssh.UnknownChannelType, "session channel required")
			continue
		}

		accepted, channelRequests, err := channel.Accept()
		if err != nil {
			return
		}
		go handleEchoSession(accepted, channelRequests)
	}
}

func handleEchoSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for request := range requests {
		if request.Type != "exec" {
			_ = request.Reply(false, nil)
			continue
		}

		_ = request.Reply(true, nil)
		// exit-status payload is a 4-byte big-endian status code.
		_, _ = channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
		return
	}
}
