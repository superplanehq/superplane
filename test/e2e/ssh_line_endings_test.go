package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	pwssh "golang.org/x/crypto/ssh"
	"gorm.io/datatypes"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	gitstorage "github.com/superplanehq/superplane/pkg/git"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"github.com/superplanehq/superplane/test/support"
)

const (
	sshLineEndingSecretName = "e2e-ssh-password"
	sshLineEndingSecretKey  = "password"
	sshLineEndingPassword   = "correct-horse-battery-staple"
	sshLineEndingUser       = "e2e"
	sshLineEndingNodeName   = "Run SSH Script"
	sshLineEndingScriptPath = "scripts/deploy.sh"
)

func TestSSHCommandLineEndings(t *testing.T) {
	t.Run("normalizes CRLF command files before streaming them over SSH", func(t *testing.T) {
		steps := &sshLineEndingsSteps{t: t}
		steps.start()
		steps.givenAnSSHServerExpectingScript("bash -s", "set -eo pipefail\nprintf ok\n")
		steps.givenACanvasWithCRLFSSHCommandFile()
		steps.whenTheManualTriggerRuns()
		steps.thenTheSSHNodeCompletedSuccessfully()
		steps.thenTheSSHServerReceivedNormalizedScript()
	})

	t.Run("normalizes CRLF inline scripts before streaming them over SSH", func(t *testing.T) {
		steps := &sshLineEndingsSteps{t: t}
		steps.start()
		steps.givenAnSSHServerExpectingScript("bash -e -s", strings.Join([]string{
			"#!/bin/bash",
			"set -euo pipefail",
			"cd ~/preview-sso.teams.novp.com",
			`sed -i -E "s#([a-z0-9\-]+\.teams\.novp\.com)#${APP_HOST_PREFIX}-\1#g" .env`,
			"docker compose down --remove-orphans && docker compose up -d",
			"docker compose run --rm php php artisan migrate",
			"",
		}, "\n"))
		steps.givenACanvasWithCRLFInlineScript()
		steps.whenTheManualTriggerRuns()
		steps.thenTheSSHNodeCompletedSuccessfully()
		steps.thenTheSSHServerReceivedNormalizedScript()
	})
}

type sshLineEndingsSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
	server  *scriptCheckingSSHServer
}

func (s *sshLineEndingsSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *sshLineEndingsSteps) givenAnSSHServerExpectingScript(expectedCommand string, expectedScript string) {
	server, err := startScriptCheckingSSHServer(sshLineEndingUser, sshLineEndingPassword, expectedCommand, expectedScript)
	require.NoError(s.t, err)
	s.t.Cleanup(server.Close)
	s.server = server
}

func (s *sshLineEndingsSteps) givenACanvasWithCRLFSSHCommandFile() {
	require.NotNil(s.t, s.server, "SSH server must be started before creating the canvas")

	s.createPasswordSecret()
	canvas := s.createPublishedSSHCanvas(map[string]any{
		"commandSource": "file",
		"commandFile":   sshLineEndingScriptPath,
	})
	s.createRepositoryCommandFile(canvas, "set -eo pipefail\r\nprintf ok\r\n")

	s.canvas = shared.NewCanvasSteps("SSH CRLF Line Endings", s.t, s.session)
	s.canvas.WorkflowID = canvas.ID
}

func (s *sshLineEndingsSteps) givenACanvasWithCRLFInlineScript() {
	require.NotNil(s.t, s.server, "SSH server must be started before creating the canvas")

	s.createPasswordSecret()
	canvas := s.createPublishedSSHCanvas(map[string]any{
		"commandSource": "inline",
		"commands": strings.Join([]string{
			"#!/bin/bash",
			"set -euo pipefail",
			"cd ~/preview-sso.teams.novp.com",
			`sed -i -E "s#([a-z0-9\-]+\.teams\.novp\.com)#${APP_HOST_PREFIX}-\1#g" .env`,
			"docker compose down --remove-orphans && docker compose up -d",
			"docker compose run --rm php php artisan migrate",
			"",
		}, "\r\n"),
	})

	s.canvas = shared.NewCanvasSteps("SSH Inline CRLF Line Endings", s.t, s.session)
	s.canvas.WorkflowID = canvas.ID
}

func (s *sshLineEndingsSteps) createPasswordSecret() {
	data, err := json.Marshal(map[string]string{
		sshLineEndingSecretKey: sshLineEndingPassword,
	})
	require.NoError(s.t, err)

	_, err = models.CreateSecret(
		sshLineEndingSecretName,
		secrets.ProviderLocal,
		s.session.Account.ID.String(),
		models.DomainTypeOrganization,
		s.session.OrgID,
		data,
	)
	require.NoError(s.t, err)
}

func (s *sshLineEndingsSteps) createPublishedSSHCanvas(commandConfig map[string]any) *models.Canvas {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	startNodeID := "start-trigger"
	sshNodeID := "ssh-command"
	configuration := map[string]any{
		"host":     "127.0.0.1",
		"port":     s.server.Port(),
		"username": sshLineEndingUser,
		"timeout":  10,
		"authentication": map[string]any{
			"authMethod": "password",
			"password": map[string]any{
				"secret": sshLineEndingSecretName,
				"key":    sshLineEndingSecretKey,
			},
		},
	}
	for key, value := range commandConfig {
		configuration[key] = value
	}

	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, []models.CanvasNode{
		{
			NodeID: startNodeID,
			Name:   "Start",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: 600, Y: 200}),
		},
		{
			NodeID: sshNodeID,
			Name:   sshLineEndingNodeName,
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "ssh"},
			}),
			Configuration: datatypes.NewJSONType(configuration),
			Position:      datatypes.NewJSONType(models.Position{X: 1000, Y: 200}),
		},
	}, []models.Edge{
		{SourceID: startNodeID, TargetID: sshNodeID, Channel: "default"},
	})

	return canvas
}

func (s *sshLineEndingsSteps) createRepositoryCommandFile(canvas *models.Canvas, script string) {
	gitProvider, err := gitstorage.NewProvider()
	require.NoError(s.t, err)

	repoID := gitProvider.GetRepositoryID(gitprovider.RepositoryOptions{
		OrganizationID: canvas.OrganizationID,
		CanvasID:       canvas.ID,
	})

	repository, err := canvas.CreatePendingRepository(gitProvider.Name(), repoID)
	require.NoError(s.t, err)

	_, err = gitProvider.CreateRepository(s.t.Context(), repoID)
	require.NoError(s.t, err)
	require.NoError(s.t, repository.MarkReady(database.Conn()))

	head, err := gitProvider.Head(s.t.Context(), repoID, "")
	require.NoError(s.t, err)

	_, err = gitProvider.Commit(s.t.Context(), repoID, gitprovider.CommitOptions{
		Branch:          "main",
		Message:         "Add SSH command file",
		ExpectedHeadSHA: head,
		Author:          gitprovider.SuperPlaneBotAuthor(),
		Operations: []gitprovider.FileOperation{
			{
				Path:      sshLineEndingScriptPath,
				Content:   strings.NewReader(script),
				SizeBytes: int64(len(script)),
			},
		},
	})
	require.NoError(s.t, err)
}

func (s *sshLineEndingsSteps) whenTheManualTriggerRuns() {
	node := s.canvas.GetNodeFromDB("Start")
	eventContext := contexts.NewEventContext(database.Conn(), node, nil, func(events []models.CanvasEvent) {
		for i := range events {
			require.NoError(s.t, messages.PublishCanvasEventCreatedMessage(&events[i]))
		}
	})

	require.NoError(s.t, eventContext.Emit("manual.run", map[string]any{"source": "e2e"}))
}

func (s *sshLineEndingsSteps) thenTheSSHNodeCompletedSuccessfully() {
	s.canvas.WaitForExecution(sshLineEndingNodeName, models.CanvasNodeExecutionStateFinished, 30*time.Second)

	executions := s.canvas.GetExecutionsForNode(sshLineEndingNodeName)
	require.NotEmpty(s.t, executions)
	execution := executions[0]
	require.Equal(s.t, models.CanvasNodeExecutionResultPassed, execution.Result)

	var successEvent models.CanvasEvent
	err := database.Conn().
		Where("workflow_id = ?", s.canvas.WorkflowID).
		Where("execution_id = ?", execution.ID).
		Where("channel = ?", "success").
		First(&successEvent).
		Error
	require.NoError(s.t, err, "expected SSH node to emit on the success channel")
}

func (s *sshLineEndingsSteps) thenTheSSHServerReceivedNormalizedScript() {
	body := s.server.WaitForScript(s.t)
	require.Equal(s.t, s.server.expected, body)
	require.NotContains(s.t, body, "\r")
}

type scriptCheckingSSHServer struct {
	listener        net.Listener
	config          *pwssh.ServerConfig
	expectedCommand string
	expected        string
	scripts         chan string
	errors          chan error
}

func startScriptCheckingSSHServer(username, password, expectedCommand, expected string) (*scriptCheckingSSHServer, error) {
	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	signer, err := pwssh.NewSignerFromKey(hostKey)
	if err != nil {
		return nil, err
	}

	config := &pwssh.ServerConfig{
		PasswordCallback: func(metadata pwssh.ConnMetadata, candidate []byte) (*pwssh.Permissions, error) {
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

	server := &scriptCheckingSSHServer{
		listener:        listener,
		config:          config,
		expectedCommand: expectedCommand,
		expected:        expected,
		scripts:         make(chan string, 1),
		errors:          make(chan error, 1),
	}

	go server.accept()
	return server, nil
}

func (s *scriptCheckingSSHServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *scriptCheckingSSHServer) Close() {
	_ = s.listener.Close()
}

func (s *scriptCheckingSSHServer) WaitForScript(t *testing.T) string {
	t.Helper()

	select {
	case script := <-s.scripts:
		return script
	case err := <-s.errors:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for SSH script")
	}

	return ""
}

func (s *scriptCheckingSSHServer) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			s.reportError(fmt.Errorf("accept SSH connection: %w", err))
			return
		}

		go s.handleConnection(conn)
	}
}

func (s *scriptCheckingSSHServer) handleConnection(conn net.Conn) {
	sshConn, channels, requests, err := pwssh.NewServerConn(conn, s.config)
	if err != nil {
		s.reportError(fmt.Errorf("handshake SSH connection: %w", err))
		return
	}
	defer sshConn.Close()

	go pwssh.DiscardRequests(requests)

	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(pwssh.UnknownChannelType, "session channel required")
			continue
		}

		accepted, channelRequests, err := channel.Accept()
		if err != nil {
			s.reportError(fmt.Errorf("accept SSH session channel: %w", err))
			return
		}

		go s.handleSession(accepted, channelRequests)
	}
}

func (s *scriptCheckingSSHServer) handleSession(channel pwssh.Channel, requests <-chan *pwssh.Request) {
	defer channel.Close()

	for request := range requests {
		if request.Type != "exec" {
			_ = request.Reply(false, nil)
			continue
		}

		var payload struct {
			Command string
		}
		if err := pwssh.Unmarshal(request.Payload, &payload); err != nil {
			_ = request.Reply(false, nil)
			s.reportError(fmt.Errorf("parse SSH exec request: %w", err))
			return
		}

		if payload.Command != s.expectedCommand {
			_ = request.Reply(false, nil)
			s.reportError(fmt.Errorf("unexpected SSH command %q", payload.Command))
			return
		}

		_ = request.Reply(true, nil)

		body, err := io.ReadAll(channel)
		if err != nil {
			s.reportError(fmt.Errorf("read SSH stdin: %w", err))
			return
		}

		status := uint32(0)
		script := string(body)
		if script != s.expected || strings.Contains(script, "\r") {
			status = 1
			_, _ = channel.Stderr().Write([]byte("script line endings were not normalized"))
		} else {
			_, _ = channel.Write([]byte("ok\n"))
		}

		s.publishScript(script)
		_, _ = channel.SendRequest("exit-status", false, pwssh.Marshal(struct {
			Status uint32
		}{Status: status}))
		return
	}
}

func (s *scriptCheckingSSHServer) publishScript(script string) {
	select {
	case s.scripts <- script:
	default:
	}
}

func (s *scriptCheckingSSHServer) reportError(err error) {
	select {
	case s.errors <- err:
	default:
	}
}
