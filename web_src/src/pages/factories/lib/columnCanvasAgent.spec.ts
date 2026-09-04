import { describe, expect, it } from "vitest";
import type { CanvasesCanvas } from "@/api-client";

import {
  applyPlanningReviewDraftToCanvas,
  canvasNodeToPlanningReviewComponent,
  findAgentNodes,
  planningReviewDraftFromCanvas,
  primaryAgentNode,
  serializeColumnAgentCanvas,
  type CanvasSpecNode,
} from "./columnCanvasAgent";

const implementerSteps = [
  { name: "Clone Repo", type: "bash", command: "git clone $REPO repo" },
  { name: "Implement", type: "prompt", prompt: "Implement the plan.", workingDirectory: "repo" },
];

const implementerConfiguration = {
  machineType: "e1-large-amd64",
  model: "sonnet",
  credentials: { source: "integration", integration: { name: "claude" } },
  environmentFrom: [{ source: "integration", integration: { name: "github" } }],
  environment: [{ name: "REPO", valueSource: "literal", value: "acme/web" }],
  steps: implementerSteps,
  executionTimeoutSeconds: 3600,
};

function agentNode(overrides: Partial<CanvasSpecNode> = {}): CanvasSpecNode {
  return {
    id: "implementation-agent",
    name: "Implement From Task Description",
    type: "TYPE_ACTION",
    component: "runnerClaudeCode",
    concurrency: { max: 5, key: "ci-{{ $.data.branch }}" },
    configuration: implementerConfiguration,
    ...overrides,
  };
}

function triggerNode(): CanvasSpecNode {
  return { id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun", configuration: {} };
}

function canvasWith(nodes: CanvasSpecNode[]): CanvasesCanvas {
  return { metadata: { id: "app-1", liveVersionId: "version-1" }, spec: { nodes, edges: [] } };
}

describe("findAgentNodes", () => {
  it("returns no nodes when the canvas has no agent harness", () => {
    expect(findAgentNodes({ nodes: [triggerNode()] })).toEqual([]);
  });

  it("returns runner harness actions and skips bash runners", () => {
    const bash: CanvasSpecNode = {
      id: "bash-1",
      type: "TYPE_ACTION",
      component: "runnerBash",
      configuration: {},
    };
    const claude = agentNode();
    expect(findAgentNodes({ nodes: [triggerNode(), bash, claude] })).toEqual([claude]);
  });

  it("skips harness components that are not actions", () => {
    const trigger = agentNode({ id: "not-an-action", type: "TYPE_TRIGGER" });
    expect(findAgentNodes({ nodes: [trigger] })).toEqual([]);
  });
});

describe("primaryAgentNode", () => {
  it("returns the first agent in canvas order", () => {
    const planner = agentNode({ id: "planner-agent", name: "Agent - Plan" });
    const implementer = agentNode();
    expect(primaryAgentNode({ nodes: [triggerNode(), planner, implementer] })?.id).toBe("planner-agent");
  });

  it("returns undefined when no agent exists", () => {
    expect(primaryAgentNode({ nodes: [triggerNode()] })).toBeUndefined();
  });
});

describe("planningReviewDraftFromCanvas", () => {
  it("maps the agent name, configuration, and concurrency without flattening integration refs", () => {
    const canvas = canvasWith([triggerNode(), agentNode()]);
    const draft = planningReviewDraftFromCanvas(canvas, "implementation-agent");

    expect(draft?.title).toBe("Implement From Task Description");
    expect(draft?.components).toHaveLength(1);
    expect(draft?.components[0].configuration).toEqual(implementerConfiguration);
    expect(draft?.components[0].configuration.credentials).toEqual({
      source: "integration",
      integration: { name: "claude" },
    });
    expect(draft?.components[0].concurrency).toEqual({ max: "5", key: "ci-{{ $.data.branch }}" });
    expect((draft?.components[0].configuration.steps as typeof implementerSteps)[0].name).toBe("Clone Repo");
  });

  it("returns null when the node id is missing", () => {
    expect(planningReviewDraftFromCanvas(canvasWith([agentNode()]), "missing")).toBeNull();
  });
});

describe("applyPlanningReviewDraftToCanvas", () => {
  it("round-trips steps, model, environment, and integration refs onto the same node", () => {
    const original = agentNode();
    const canvas = canvasWith([triggerNode(), original]);
    const draft = planningReviewDraftFromCanvas(canvas, "implementation-agent");
    if (!draft) {
      throw new Error("expected a draft");
    }
    draft.components[0].title = "Agent - Implement";
    draft.components[0].configuration = {
      ...draft.components[0].configuration,
      model: "opus",
    };
    draft.components[0].concurrency = { max: "8", key: "" };

    const next = applyPlanningReviewDraftToCanvas(canvas, "implementation-agent", draft);
    const patched = next.spec?.nodes?.find((node) => node.id === "implementation-agent");
    const untouched = next.spec?.nodes?.find((node) => node.id === "on-run");

    expect(untouched).toEqual(triggerNode());
    expect(patched?.name).toBe("Agent - Implement");
    expect(patched?.configuration).toMatchObject({
      model: "opus",
      credentials: { source: "integration", integration: { name: "claude" } },
      environment: [{ name: "REPO", valueSource: "literal", value: "acme/web" }],
      steps: implementerSteps,
    });
    expect(patched?.concurrency).toEqual({ max: 8 });
  });

  it("serializes the patched agent into canvas.yaml", () => {
    const canvas = canvasWith([triggerNode(), agentNode()]);
    const draft = planningReviewDraftFromCanvas(canvas, "implementation-agent");
    if (!draft) {
      throw new Error("expected a draft");
    }
    draft.components[0].configuration = {
      ...draft.components[0].configuration,
      model: "opus",
    };

    const yamlText = serializeColumnAgentCanvas(canvas, "implementation-agent", draft);

    expect(yamlText).toContain("kind: Canvas");
    expect(yamlText).toContain("opus");
    expect(yamlText).toContain("Clone Repo");
    expect(yamlText).toContain("name: claude");
  });
});

describe("canvasNodeToPlanningReviewComponent", () => {
  it("defaults a missing name and concurrency", () => {
    const component = canvasNodeToPlanningReviewComponent({
      id: "agent-1",
      type: "TYPE_ACTION",
      component: "runnerCodex",
    });
    expect(component.title).toBe("Agent");
    expect(component.concurrency).toEqual({ max: "1", key: "" });
  });
});
