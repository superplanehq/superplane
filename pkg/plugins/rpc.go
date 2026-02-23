package plugins

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

const DefaultExecutionTimeout = 30 * time.Second
const DefaultActivationTimeout = 10 * time.Second

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ContextCallbackHandler is called when the Plugin Host sends a request
// back to Go for context operations (secrets, http, metadata, etc.).
type ContextCallbackHandler func(method string, params json.RawMessage) (any, error)

type PluginHostProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	nextID  atomic.Int64
	pending sync.Map // map[int64]chan *jsonRPCResponse
	writeMu sync.Mutex

	callbackHandler ContextCallbackHandler

	done chan struct{}
}

func SpawnPluginHost(pluginHostPath string, pluginsDir string, callbackHandler ContextCallbackHandler) (*PluginHostProcess, error) {
	cmd := exec.Command("node", pluginHostPath)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("SUPERPLANE_PLUGINS_DIR=%s", pluginsDir))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting plugin host: %w", err)
	}

	p := &PluginHostProcess{
		cmd:             cmd,
		stdin:           stdin,
		stdout:          stdout,
		stderr:          stderr,
		callbackHandler: callbackHandler,
		done:            make(chan struct{}),
	}

	go p.readLoop()
	go p.readStderr()

	return p, nil
}

func (p *PluginHostProcess) readLoop() {
	defer close(p.done)
	scanner := bufio.NewScanner(p.stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		var msg json.RawMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.WithError(err).Warn("Plugin Host: invalid JSON from stdout")
			continue
		}

		// Determine if this is a response (has "id" + ("result" or "error"))
		// or a request from the Plugin Host (has "method").
		var peek struct {
			ID     *int64          `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Error  *jsonRPCError   `json:"error"`
		}
		if err := json.Unmarshal(line, &peek); err != nil {
			continue
		}

		if peek.Method != "" && peek.ID != nil {
			// This is a request FROM the Plugin Host (e.g., ctx/secrets.getKey)
			go p.handleCallback(*peek.ID, peek.Method, line)
			continue
		}

		if peek.ID != nil {
			// This is a response to one of our requests
			var resp jsonRPCResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				log.WithError(err).Warn("Plugin Host: failed to parse response")
				continue
			}

			if ch, ok := p.pending.LoadAndDelete(resp.ID); ok {
				ch.(chan *jsonRPCResponse) <- &resp
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Error("Plugin Host stdout reader error")
	}
}

func (p *PluginHostProcess) readStderr() {
	scanner := bufio.NewScanner(p.stderr)
	for scanner.Scan() {
		log.WithField("source", "plugin-host").Warn(scanner.Text())
	}
}

func (p *PluginHostProcess) handleCallback(id int64, method string, raw []byte) {
	var req struct {
		Params json.RawMessage `json:"params"`
	}
	_ = json.Unmarshal(raw, &req)

	result, err := p.callbackHandler(method, req.Params)

	var resp jsonRPCResponse
	resp.JSONRPC = "2.0"
	resp.ID = id

	if err != nil {
		resp.Error = &jsonRPCError{
			Code:    -32000,
			Message: err.Error(),
		}
	} else {
		resultBytes, _ := json.Marshal(result)
		resp.Result = resultBytes
	}

	p.writeMessage(resp)
}

func (p *PluginHostProcess) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := p.nextID.Add(1)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	respCh := make(chan *jsonRPCResponse, 1)
	p.pending.Store(id, respCh)
	defer p.pending.Delete(id)

	if err := p.writeMessage(req); err != nil {
		return nil, fmt.Errorf("writing to plugin host: %w", err)
	}

	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("plugin error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.done:
		return nil, fmt.Errorf("plugin host process exited")
	}
}

func (p *PluginHostProcess) writeMessage(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	data = append(data, '\n')
	_, err = p.stdin.Write(data)
	return err
}

func (p *PluginHostProcess) Kill() {
	_ = p.stdin.Close()
	_ = p.cmd.Process.Kill()
	_ = p.cmd.Wait()
}

func (p *PluginHostProcess) Wait() error {
	return p.cmd.Wait()
}

func (p *PluginHostProcess) Done() <-chan struct{} {
	return p.done
}
