package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
)

const getSkillActionName = "get_skill"

type getSkillAction struct{}

func (getSkillAction) Name() string {
	return getSkillActionName
}

func (getSkillAction) Execute(_ context.Context, _ agents.AgentSessionContext, input Input) (any, error) {
	skill := strings.TrimSpace(input.Skill)
	if skill == "" {
		return nil, fmt.Errorf("skill is required for get_skill")
	}

	switch skill {
	case "console_yaml":
		return consoleYAMLSkill(), nil
	default:
		return nil, fmt.Errorf("unsupported skill %q", input.Skill)
	}
}

func consoleYAMLSkill() skillResult {
	return skillResult{
		Action: getSkillActionName,
		Skill:  "console_yaml",
		Title:  "Strict Console YAML",
		Body: `Use this exact envelope for console_yaml:

apiVersion: v1
kind: Console
metadata:
  name: Ops Console
spec:
  panels:
    - id: intro
      type: markdown
      content:
        body: "# Hello"
  layout:
    - i: intro
      x: 0
      y: 0
      w: 12
      h: 4

Rules:
- kind must be Console, never Dashboard.
- name is allowed only as metadata.name. Never use root name, panel name, or layout name.
- spec may contain only panels and layout.
- panel objects may contain only id, type, content.
- layout items use i, x, y, w, h, optional minW, minH. i must match a panel id.

Node panels:
- type: node shows one node with content.node: "<canvas node id or name>".
- type: nodes shows multiple nodes with content.nodes as an array of objects.
- each content.nodes item must include node.

Valid nodes panel:

- id: nodes-overview
  type: nodes
  content:
    title: Key Nodes
    nodes:
      - node: start
        label: Manual Start
        description: Starts the flow
        showRun: true
      - node: deploy-prod
        description: Deploys the latest build
`,
		Notes: []string{
			"If validation says unknown field \"name\", move it to metadata.name if it is the console title; otherwise remove it.",
			"If validation says content.nodes must be an array, use nodes: [{node: \"...\"}] YAML form, not an object or string list.",
			"If validation says content.nodes[0].node is required, each nodes entry needs node set to a canvas node id or name.",
		},
	}
}
