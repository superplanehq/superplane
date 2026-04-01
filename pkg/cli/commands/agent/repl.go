package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/cli/commands/agent/widgets"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	"golang.org/x/term"
)

var chatStreamPathPattern = regexp.MustCompile(`/agents/chats/([^/]+)/stream/?$`)
var exitCommands = []string{"/exit", "/quit", "exit", "quit"}

type ReplOptions struct {
	Canvas *openapi_client.CanvasesCanvas
}

type Repl struct {
	options        ReplOptions
	out            io.Writer
	currentSession *ChatSession
	result         *PromptResult
	waitingStatus  *TransientStatus
}

type PromptResult struct {
	Model       string
	Streamed    bool
	FinalAnswer *FinalAnswer
}

func NewRepl(options ReplOptions) *Repl {
	return &Repl{
		options: options,
	}
}

func (r *Repl) Run(ctx core.CommandContext) error {
	r.out = ctx.Cmd.OutOrStdout()

	if err := r.renderHeader(); err != nil {
		return err
	}

	var nextPrompt string

	for {

		//
		// If we don't have a prompt, we need to prompt the user for input.
		//
		if nextPrompt == "" {
			prompt, canceled, err := widgets.NewPrompt(r.out).Run()
			if err != nil {
				return err
			}

			if canceled {
				return nil
			}

			nextPrompt = prompt
		}

		if nextPrompt == "" {
			continue
		}

		//
		// If the prompt is an exit command, we exit the REPL.
		//
		if slices.Contains(exitCommands, strings.ToLower(nextPrompt)) {
			return nil
		}

		//
		// Otherwise, we run the prompt through the agent.
		//
		currentSession, err := r.createChatSession(ctx)
		if err != nil {
			return err
		}

		r.currentSession = currentSession
		err = r.runPrompt(ctx, nextPrompt)
		if err != nil {
			return err
		}

		if r.result.FinalAnswer != nil && r.result.FinalAnswer.Proposal != nil {
			if err := r.renderProposal(); err != nil {
				return err
			}

			picker := widgets.NewPicker(r.out, []widgets.PickerOption{
				{
					ID:    "apply",
					Title: "Apply",
				},
				{
					ID:    "reject",
					Title: "Reject",
				}},
			)

			action, canceled, err := picker.Run()
			if err != nil {
				return err
			}

			if canceled {
				return nil
			}

			switch action {
			case "apply":
				if err := r.applyPendingProposal(ctx); err != nil {
					return err
				}
			case "reject":
				r.result.FinalAnswer = nil
				if err := r.renderStatusLine("Proposal discarded."); err != nil {
					return err
				}
			}
		}

		nextPrompt = ""
	}
}

func (r *Repl) createChatSession(ctx core.CommandContext) (*ChatSession, error) {
	body := openapi_client.NewAgentsCreateAgentChatRequest()
	body.SetCanvasId(r.options.Canvas.Metadata.GetId())

	response, _, err := ctx.API.AgentAPI.
		AgentsCreateAgentChat(ctx.Context).
		Body(*body).
		Execute()

	if err != nil {
		return nil, err
	}

	chatID, err := r.extractChatIDFromURL(response.GetUrl())
	if err != nil {
		return nil, err
	}

	return &ChatSession{
		ChatID: chatID,
		Token:  strings.TrimSpace(response.GetToken()),
		URL:    strings.TrimSpace(response.GetUrl()),
	}, nil
}

func (r *Repl) extractChatIDFromURL(raw string) (string, error) {
	match := chatStreamPathPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) < 2 || strings.TrimSpace(match[1]) == "" {
		return "", fmt.Errorf("invalid chat stream url %q", raw)
	}

	return strings.TrimSpace(match[1]), nil
}

func (r *Repl) renderHeader() error {
	_, err := fmt.Fprintf(
		r.out,
		"Canvas: %s (%s)\n",
		*r.options.Canvas.Metadata.Name,
		r.options.Canvas.Metadata.GetId(),
	)

	return err
}

func (r *Repl) renderPrompt(prompt string) error {
	lines := strings.Split(strings.TrimSpace(prompt), "\n")
	if _, err := fmt.Fprintln(r.out); err != nil {
		return err
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(r.out, "> %s\n", line); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repl) runPrompt(ctx core.CommandContext, prompt string) error {
	if err := r.renderPrompt(prompt); err != nil {
		return err
	}

	//
	// Start a transient status to show the user that the stream is ongoing.
	//
	r.waitingStatus = NewTransientStatus(r.out, "Planning next steps...")
	defer r.waitingStatus.Stop()

	r.result = &PromptResult{}

	//
	// Send prompt to agent and stream events to the REPL.
	//
	err := r.currentSession.Stream(ctx, prompt, r.onStreamEvent)
	if err != nil {
		return err
	}

	r.waitingStatus.Stop()

	if strings.TrimSpace(r.result.FinalAnswer.Answer) != "" {
		if err := r.renderAssistantMarkdown(r.terminalWidth(), r.result.FinalAnswer.Answer); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repl) onStreamEvent(event ChatStreamEvent) error {
	switch event.Type {
	case "run_started":
		r.result.Model = *event.Model
		return nil

	case "model_delta":
		if event.Content == nil || *event.Content == "" {
			return nil
		}

		r.result.Streamed = true
		if r.result.FinalAnswer != nil {
			r.result.FinalAnswer.Answer += *event.Content
		} else {
			r.result.FinalAnswer = &FinalAnswer{
				Answer: *event.Content,
			}
		}

		return nil

	case "tool_started", "tool_finished":
		return r.waitingStatus.WriteLine(func() error {
			return r.renderToolActivity(event)
		})

	case "final_answer":
		return r.parseFinalAnswer(event)

	case "run_failed":
		if event.Error == nil || strings.TrimSpace(*event.Error) == "" {
			return fmt.Errorf("agent run failed")
		}

		return fmt.Errorf("%s", strings.TrimSpace(*event.Error))
	}

	return nil
}

type FinalAnswer struct {
	Answer   string    `json:"answer"`
	Proposal *Proposal `json:"proposal"`
}

type Proposal struct {
	ID         string              `json:"id"`
	Summary    string              `json:"summary"`
	Operations []ProposalOperation `json:"operations"`
}

type ProposalOperation struct {
	Type          string         `json:"type"`
	BlockName     string         `json:"block_name"`
	NodeKey       string         `json:"node_key"`
	NodeName      string         `json:"node_name"`
	Configuration map[string]any `json:"configuration"`
	Position      *Position      `json:"position"`
	Source        *NodeRef       `json:"source"`
	Target        *NodeRef       `json:"target"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type NodeRef struct {
	NodeKey     string `json:"node_key"`
	NodeID      string `json:"node_id"`
	NodeName    string `json:"node_name"`
	HandleID    string `json:"handle_id"`
	HasHandleID bool   `json:"has_handle_id"`
}

func (r *Repl) parseFinalAnswer(event ChatStreamEvent) error {
	if event.Output == nil {
		return nil
	}

	finalAnswer := FinalAnswer{}
	err := mapstructure.Decode(*event.Output, &finalAnswer)
	if err != nil {
		return err
	}

	r.result.FinalAnswer = &finalAnswer
	return nil
}

func (r *Repl) renderProposal() error {
	lines := []string{r.result.FinalAnswer.Proposal.Summary}
	for _, summary := range r.result.FinalAnswer.Proposal.SummaryLines() {
		lines = append(lines, "- "+summary)
	}

	block := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "33", Dark: "75"}).
		Padding(0, 1).
		Width(max(28, r.terminalWidth()-2)).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	_, err := fmt.Fprintf(r.out, "\n%s\n", block)
	return err
}

func (r *Repl) applyPendingProposal(ctx core.CommandContext) error {
	if r.result.FinalAnswer.Proposal == nil {
		return r.renderStatusLine("No pending proposal.")
	}

	err := r.result.FinalAnswer.Proposal.Apply(ctx, *r.options.Canvas)
	if err != nil {
		return err
	}

	r.result.FinalAnswer.Proposal = nil
	return r.renderStatusLine("Applied the proposed changes to the canvas.")
}

func (r *Repl) renderStatusLine(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	_, err := fmt.Fprintf(r.out, "%s\n", content)
	return err
}

func (r *Repl) renderToolActivity(event ChatStreamEvent) error {
	toolLabel := r.formatToolLabel(event.ToolName)

	style := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "242", Dark: "244"})

	switch event.Type {
	case "tool_started":
		style = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "166", Dark: "214"})

	case "tool_finished":
		if event.ElapsedMS != nil {
			toolLabel += fmt.Sprintf(" (%.1fms)", *event.ElapsedMS)
		}

		style = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "203"})
	}

	_, err := fmt.Fprintf(r.out, "%s\n", style.Render("[tool] "+toolLabel))
	return err
}

func (r *Repl) renderAssistantMarkdown(width int, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(20, width-2)),
	)
	if err != nil {
		_, err = fmt.Fprintf(r.out, "\n%s\n", content)
		return err
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		_, err = fmt.Fprintf(r.out, "\n%s\n", content)
		return err
	}

	_, err = fmt.Fprintf(r.out, "\n%s\n", strings.TrimRight(rendered, "\n"))
	return err
}

func (r *Repl) terminalWidth() int {
	file, ok := r.out.(*os.File)
	if !ok {
		return 100
	}

	width, _, err := term.GetSize(int(file.Fd()))
	if err != nil || width <= 0 {
		return 100
	}

	return width
}

func (r *Repl) formatToolLabel(toolName *string) string {
	if toolName == nil {
		return "Running tool"
	}

	normalized := strings.ToLower(strings.TrimSpace(*toolName))
	switch normalized {
	case "get_canvas_shape":
		return "Reading canvas structure"
	case "get_canvas_details":
		return "Reading canvas details"
	case "list_available_blocks":
		return "Listing available components"
	}

	replaced := strings.Join(strings.Fields(strings.NewReplacer("_", " ", "-", " ").Replace(normalized)), " ")
	if replaced == "" {
		return "Running tool"
	}

	return strings.ToUpper(replaced[:1]) + replaced[1:]
}

type TransientStatus struct {
	writer  io.Writer
	message string

	mu      sync.Mutex
	frame   int
	visible bool
	stopped bool
	done    chan struct{}
	once    sync.Once
}

func NewTransientStatus(writer io.Writer, message string) *TransientStatus {
	return &TransientStatus{
		writer:  writer,
		message: strings.TrimSpace(message),
		done:    make(chan struct{}),
	}
}

func (s *TransientStatus) Start(writer io.Writer, message string) {
	s.render()
	go s.loop()
}

func (s *TransientStatus) loop() {
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.render()
		case <-s.done:
			return
		}
	}
}

func (s *TransientStatus) Stop() {
	s.once.Do(func() {
		close(s.done)

		s.mu.Lock()
		defer s.mu.Unlock()

		s.stopped = true
		s.clearLocked()
	})
}

func (s *TransientStatus) WriteLine(write func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return write()
	}

	s.clearLocked()
	if err := write(); err != nil {
		return err
	}

	s.renderLocked()
	return nil
}

func (s *TransientStatus) render() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	s.renderLocked()
}

func (s *TransientStatus) renderLocked() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[s.frame%len(frames)]
	s.frame++
	_, _ = fmt.Fprintf(s.writer, "\r\033[2K%s %s", frame, s.message)
	s.visible = true
}

func (s *TransientStatus) clearLocked() {
	if !s.visible {
		return
	}

	_, _ = fmt.Fprint(s.writer, "\r\033[2K\r")
	s.visible = false
}

type ChatSession struct {
	ChatID string
	Token  string
	URL    string
}

type ChatStreamEvent struct {
	Type       string   `json:"type"`
	Model      *string  `json:"model"`
	Content    *string  `json:"content"`
	ToolName   *string  `json:"tool_name"`
	ToolCallID *string  `json:"tool_call_id"`
	ElapsedMS  *float64 `json:"elapsed_ms"`
	Output     *any     `json:"output"`
	Error      *string  `json:"error"`
}

func (c *ChatSession) Stream(ctx core.CommandContext, prompt string, onEvent func(ChatStreamEvent) error) error {
	requestBody, err := json.Marshal(map[string]string{
		"question": prompt,
	})

	if err != nil {
		return fmt.Errorf("failed to encode prompt: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx.Context, http.MethodPost, c.URL, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create agent chat request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", "Bearer "+c.Token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to request agent chat stream: %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode >= 300 {
		payload, _ := io.ReadAll(response.Body)
		message := strings.TrimSpace(string(payload))
		if message == "" {
			message = response.Status
		}

		return fmt.Errorf("agent chat request failed: %s", message)
	}

	return c.stream(response.Body, onEvent)
}

func (c *ChatSession) stream(reader io.Reader, onEvent func(ChatStreamEvent) error) error {
	bufferedReader := bufio.NewReader(reader)
	dataLines := make([]string, 0, 4)

	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}

		joined := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = dataLines[:0]
		if joined == "" {
			return nil
		}

		event := ChatStreamEvent{}
		if err := json.Unmarshal([]byte(joined), &event); err != nil {
			return nil
		}

		return onEvent(event)
	}

	for {
		line, err := bufferedReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read agent stream: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if flushErr := flush(); flushErr != nil {
				return flushErr
			}
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if err == io.EOF {
			return flush()
		}
	}
}

func (p *Proposal) SummaryLines() []string {
	lines := make([]string, 0, len(p.Operations))
	for _, operation := range p.Operations {
		if operation.Type == "connect_nodes" {
			continue
		}

		lines = append(lines, p.formatOperation(operation))
	}

	if len(lines) == 0 {
		for _, operation := range p.Operations {
			lines = append(lines, p.formatOperation(operation))
		}
	}

	return lines
}

// TODO: refactor this
func (p *Proposal) formatOperation(operation ProposalOperation) string {
	operationNodeLabels := make(map[string]string)
	for _, item := range p.Operations {
		if item.Type == "add_node" && strings.TrimSpace(item.NodeKey) != "" {
			label := strings.TrimSpace(item.NodeName)
			if label == "" {
				label = strings.TrimSpace(item.BlockName)
			}
			if label != "" {
				operationNodeLabels[item.NodeKey] = label
			}
		}
	}

	resolveRefLabel := func(ref *NodeRef) string {
		if ref == nil {
			return "step"
		}
		if ref.NodeName != "" {
			return ref.NodeName
		}
		if ref.NodeKey != "" {
			if label := strings.TrimSpace(operationNodeLabels[ref.NodeKey]); label != "" {
				return label
			}
		}
		if ref.NodeID != "" {
			return ref.NodeID
		}
		return "step"
	}

	switch operation.Type {
	case "add_node":
		name := strings.TrimSpace(operation.NodeName)
		if name == "" {
			name = strings.TrimSpace(operation.BlockName)
		}
		return fmt.Sprintf("Add node %s (%s)", name, operation.BlockName)

	case "connect_nodes":
		return fmt.Sprintf("Connect %s -> %s", resolveRefLabel(operation.Source), resolveRefLabel(operation.Target))

	case "disconnect_nodes":
		return fmt.Sprintf("Disconnect %s -> %s", resolveRefLabel(operation.Source), resolveRefLabel(operation.Target))

	case "update_node_config":
		name := strings.TrimSpace(operation.NodeName)
		if name == "" && operation.Target != nil {
			name = strings.TrimSpace(operation.Target.NodeName)
		}
		if name == "" {
			name = "node"
		}
		return fmt.Sprintf("Update configuration for %s", name)

	case "delete_node":
		return fmt.Sprintf("Delete node %s", resolveRefLabel(operation.Target))
	}
	return "Update canvas"
}

func (p *Proposal) Apply(ctx core.CommandContext, canvas openapi_client.CanvasesCanvas) error {
	components, _, err := ctx.API.ComponentAPI.ComponentsListComponents(ctx.Context).Execute()
	if err != nil {
		return err
	}

	triggers, _, err := ctx.API.TriggerAPI.TriggersListTriggers(ctx.Context).Execute()
	if err != nil {
		return err
	}

	updater := NewCanvasUpdater(canvas, components.GetComponents(), triggers.GetTriggers())
	newCanvas, err := updater.Apply(p)
	if err != nil {
		return err
	}

	body := openapi_client.NewCanvasesUpdateCanvasVersionBody()
	autoLayout := openapi_client.NewCanvasesCanvasAutoLayout()
	autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
	autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
	body.SetAutoLayout(*autoLayout)
	body.SetCanvas(*newCanvas)

	_, _, err = ctx.API.CanvasVersionAPI.
		CanvasesUpdateCanvasVersion2(ctx.Context, canvas.Metadata.GetId()).
		Body(*body).
		Execute()

	return err
}

type CanvasUpdater struct {
	Canvas     *openapi_client.CanvasesCanvas
	Components []openapi_client.ComponentsComponent
	Triggers   []openapi_client.TriggersTrigger

	createdNodeIDsByKey map[string]string
}

func NewCanvasUpdater(canvas openapi_client.CanvasesCanvas, components []openapi_client.ComponentsComponent, triggers []openapi_client.TriggersTrigger) *CanvasUpdater {
	return &CanvasUpdater{
		Canvas:              &canvas,
		Components:          components,
		Triggers:            triggers,
		createdNodeIDsByKey: map[string]string{},
	}
}

func (u *CanvasUpdater) Apply(proposal *Proposal) (*openapi_client.CanvasesCanvas, error) {
	for _, operation := range proposal.Operations {
		switch operation.Type {
		case "add_node":
			if err := u.addNode(operation); err != nil {
				return nil, err
			}

		case "connect_nodes":
			if err := u.connectNodes(operation); err != nil {
				return nil, err
			}

		case "disconnect_nodes":
			if err := u.disconnectNodes(operation); err != nil {
				return nil, err
			}

		case "update_node_config":
			if err := u.updateNodeConfig(operation); err != nil {
				return nil, err
			}

		case "delete_node":
			if err := u.deleteNode(operation); err != nil {
				return nil, err
			}
		}
	}

	return u.Canvas, nil
}

func (u *CanvasUpdater) addNode(operation ProposalOperation) error {
	node := openapi_client.NewComponentsNode()
	nodeID := uuid.NewString()
	node.SetId(nodeID)
	node.SetName(uniqueNodeName(u.Canvas.Spec.Nodes, operation))
	u.setNodePosition(node, operation)
	u.setNodeRef(node, operation)

	//
	// Add node to canvas spec.
	//
	u.Canvas.Spec.Nodes = append(u.Canvas.Spec.Nodes, *node)
	if operation.NodeKey != "" {
		u.createdNodeIDsByKey[operation.NodeKey] = nodeID
	}

	//
	// Add edge to canvas spec if source node is provided.
	//
	sourceID := u.resolveNode(operation.Source)
	if sourceID != "" {
		u.Canvas.Spec.Edges = appendEdgeIfMissing(u.Canvas.Spec.Edges, sourceID, nodeID, operation.Source)
	}

	return nil
}

func (u *CanvasUpdater) availableComponent(blockName string) bool {
	for _, component := range u.Components {
		if component.GetName() == blockName {
			return true
		}
	}
	return false
}

func (u *CanvasUpdater) availableTrigger(blockName string) bool {
	for _, trigger := range u.Triggers {
		if trigger.GetName() == blockName {
			return true
		}
	}
	return false
}

func (u *CanvasUpdater) setNodePosition(node *openapi_client.ComponentsNode, operation ProposalOperation) error {
	position := openapi_client.NewComponentsPosition()
	if operation.Position != nil {
		position.SetX(int32(operation.Position.X))
		position.SetY(int32(operation.Position.Y))
		node.SetPosition(*position)
		return nil
	}

	position.SetX(int32(160 * len(u.Canvas.Spec.Nodes)))
	position.SetY(80)
	node.SetPosition(*position)
	return nil
}

// TODO: it would be better if the model returned a blockType field too.
func (u *CanvasUpdater) setNodeRef(node *openapi_client.ComponentsNode, operation ProposalOperation) error {
	if u.availableComponent(operation.BlockName) {
		node.SetType(openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT)
		component := openapi_client.NewNodeComponentRef()
		component.SetName(operation.BlockName)
		node.SetComponent(*component)
		return nil
	}

	if u.availableTrigger(operation.BlockName) {
		node.SetType(openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER)
		trigger := openapi_client.NewNodeTriggerRef()
		trigger.SetName(operation.BlockName)
		node.SetTrigger(*trigger)
		return nil
	}

	return fmt.Errorf("unsupported block type for %q", operation.BlockName)
}

func (u *CanvasUpdater) connectNodes(operation ProposalOperation) error {
	sourceID := u.resolveNode(operation.Source)
	targetID := u.resolveNode(operation.Target)
	if sourceID == "" || targetID == "" {
		return fmt.Errorf("proposal references unknown nodes in connect operation")
	}

	u.Canvas.Spec.Edges = appendEdgeIfMissing(u.Canvas.Spec.Edges, sourceID, targetID, operation.Source)
	return nil
}

func (u *CanvasUpdater) disconnectNodes(operation ProposalOperation) error {
	sourceID := u.resolveNode(operation.Source)
	targetID := u.resolveNode(operation.Target)
	if sourceID == "" || targetID == "" {
		return fmt.Errorf("proposal references unknown nodes in disconnect operation")
	}

	filtered := make([]openapi_client.ComponentsEdge, 0, len(u.Canvas.Spec.Edges))
	for _, edge := range u.Canvas.Spec.Edges {
		if edge.GetSourceId() != sourceID || edge.GetTargetId() != targetID {
			filtered = append(filtered, edge)
			continue
		}

		if operation.Source == nil || !operation.Source.HasHandleID {
			continue
		}
		if edge.GetChannel() == operation.Source.HandleID {
			continue
		}

		filtered = append(filtered, edge)
	}

	u.Canvas.Spec.Edges = filtered
	return nil
}

func (u *CanvasUpdater) updateNodeConfig(operation ProposalOperation) error {
	index := u.findNodeIndex(operation.Target)
	if index < 0 {
		return fmt.Errorf("proposal references unknown node in update operation")
	}

	node := u.Canvas.Spec.Nodes[index]
	for key, value := range operation.Configuration {
		node.Configuration[key] = value
	}

	if strings.TrimSpace(operation.NodeName) != "" {
		node.SetName(uniqueNodeNameExcept(u.Canvas.Spec.Nodes, operation.NodeName, index))
	}

	u.Canvas.Spec.Nodes[index] = node
	return nil
}

func (u *CanvasUpdater) deleteNode(operation ProposalOperation) error {
	index := u.findNodeIndex(operation.Target)
	if index < 0 {
		return fmt.Errorf("proposal references unknown node in delete operation")
	}

	nodeID := u.Canvas.Spec.Nodes[index].GetId()
	u.Canvas.Spec.Nodes = append(u.Canvas.Spec.Nodes[:index], u.Canvas.Spec.Nodes[index+1:]...)

	filtered := make([]openapi_client.ComponentsEdge, 0, len(u.Canvas.Spec.Edges))
	for _, edge := range u.Canvas.Spec.Edges {
		if edge.GetSourceId() == nodeID || edge.GetTargetId() == nodeID {
			continue
		}

		filtered = append(filtered, edge)
	}

	u.Canvas.Spec.Edges = filtered
	return nil
}

func (u *CanvasUpdater) resolveNode(ref *NodeRef) string {
	if ref == nil {
		return ""
	}

	if ref.NodeKey != "" {
		if id := strings.TrimSpace(u.createdNodeIDsByKey[ref.NodeKey]); id != "" {
			return id
		}
	}

	if ref.NodeID != "" {
		return ref.NodeID
	}

	if ref.NodeName != "" {
		for _, node := range u.Canvas.Spec.Nodes {
			if node.GetId() == ref.NodeName || node.GetName() == ref.NodeName {
				return node.GetId()
			}
		}
	}

	return ""
}

func (u *CanvasUpdater) findNodeIndex(ref *NodeRef) int {
	targetID := u.resolveNode(ref)
	if targetID == "" {
		return -1
	}

	for i, node := range u.Canvas.Spec.Nodes {
		if node.GetId() == targetID {
			return i
		}
	}

	return -1
}

func appendEdgeIfMissing(edges []openapi_client.ComponentsEdge, sourceID string, targetID string, sourceRef *NodeRef) []openapi_client.ComponentsEdge {
	channel := ""
	if sourceRef != nil && sourceRef.HasHandleID {
		channel = sourceRef.HandleID
	}

	for _, edge := range edges {
		if edge.GetSourceId() == sourceID && edge.GetTargetId() == targetID && edge.GetChannel() == channel {
			return edges
		}
	}

	edge := openapi_client.NewComponentsEdge()
	edge.SetSourceId(sourceID)
	edge.SetTargetId(targetID)
	if channel != "" {
		edge.SetChannel(channel)
	}

	return append(edges, *edge)
}

func uniqueNodeName(nodes []openapi_client.ComponentsNode, operation ProposalOperation) string {
	name := strings.TrimSpace(operation.NodeName)
	if name == "" {
		name = strings.TrimSpace(operation.BlockName)
	}
	if name == "" {
		name = "node"
	}

	return uniqueNodeNameExcept(nodes, name, -1)
}

func uniqueNodeNameExcept(nodes []openapi_client.ComponentsNode, base string, skipIndex int) string {
	name := strings.TrimSpace(base)
	if name == "" {
		name = "node"
	}

	existing := make(map[string]struct{}, len(nodes))
	for i, node := range nodes {
		if i == skipIndex {
			continue
		}
		if nodeName := strings.TrimSpace(node.GetName()); nodeName != "" {
			existing[strings.ToLower(nodeName)] = struct{}{}
		}
	}

	if _, ok := existing[strings.ToLower(name)]; !ok {
		return name
	}

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s %d", name, i)
		if _, ok := existing[strings.ToLower(candidate)]; !ok {
			return candidate
		}
	}
}
