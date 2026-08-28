import { describe, expect, it } from "vitest";
import yaml from "js-yaml";

import { getFactoryDefinition, ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS } from "./index";
import { FACTORY_CANVAS_ID_PLACEHOLDER, materializeFactoryCanvas } from "./materializeFactoryTemplate";

const AGENT_COMPONENTS = ["runnerClaudeCode", "runnerCodex", "runnerOpenRouter"];

type AgentStep = { name?: string; command?: string; workingDirectory?: string };

type CanvasNode = {
  id?: string;
  component?: string;
  concurrency?: { max?: number };
  configuration?: { model?: string; steps?: AgentStep[] };
};

function canvasNodes(canvasYaml: string): CanvasNode[] {
  const doc = yaml.load(canvasYaml) as { spec?: { nodes?: CanvasNode[] } };
  return doc.spec?.nodes ?? [];
}

function nodeStepsByName(nodes: CanvasNode[], nodeId: string): Record<string, AgentStep> {
  const node = nodes.find((candidate) => candidate.id === nodeId);
  const steps = node?.configuration?.steps ?? [];
  return Object.fromEntries(steps.map((step) => [step.name ?? "", step]));
}

function materializeOnboardingApp(factoryId: string) {
  return materializeFactoryCanvas({
    definition: getFactoryDefinition(factoryId),
    canvasName: "My Planning",
    canvasId: "canvas-abc",
    installParams: { appRepository: "acme/app", backlogRepository: "acme/backlog" },
    integrations: {
      github: { id: "int-1", name: "acme-github", ready: true },
      claude: { id: "int-2", name: "acme-claude", ready: true },
    },
  });
}

describe("setup factory line apps", () => {
  it("installs plan and implement only, and leaves verify out of the workspace", () => {
    expect(ONBOARDING_LINE_APPS.map((app) => app.factoryId)).toEqual(["line-planning", "line-implementation"]);
    expect(ONBOARDING_LINE_APPS.map((app) => app.factoryId)).not.toContain("line-pr");
  });

  it("exposes a single onRun entrypoint per installed app", () => {
    for (const app of ONBOARDING_LINE_APPS) {
      const canvasYaml = materializeOnboardingApp(app.factoryId);
      expect(canvasYaml).toMatch(new RegExp(`id: ${app.entrypointNodeId}[\\s\\S]*component: onRun`));
    }
  });

  it("materializes repositories and integration wiring, leaving no template placeholders", () => {
    for (const app of ONBOARDING_LINE_APPS) {
      const canvasYaml = materializeOnboardingApp(app.factoryId);
      expect(canvasYaml).toContain("name: acme-claude");
      expect(canvasYaml).toContain("name: acme-github");
      expect(canvasYaml).not.toContain("{{ install_params.");
      expect(canvasYaml).not.toContain(FACTORY_CANVAS_ID_PLACEHOLDER);
      // No org-specific bindings copied from the production canvases.
      expect(canvasYaml).not.toContain("superplanehq");
      expect(canvasYaml).not.toContain("claude-superplane-apps");
    }
  });

  it("gives the planning agent the deeper model the rewrite resolved", () => {
    const planning = materializeOnboardingApp("line-planning");
    const implementation = materializeOnboardingApp("line-implementation");
    const planningAgent = canvasNodes(planning).find((node) => node.id === "planner-agent-no-issue");
    const implementationAgent = canvasNodes(implementation).find((node) => node.id === "implementation-agent-no-issue");

    expect(planningAgent?.configuration?.model).toBe("opus");
    expect(implementationAgent?.configuration?.model).toBe("sonnet");

    // A hosted run rejects a model that is not on the allowlist, so the
    // planner takes the resolved Opus id rather than the "opus" alias.
    const rewrittenPlanning = materializeFactoryCanvas({
      definition: getFactoryDefinition("line-planning"),
      canvasName: "Plan",
      canvasId: "canvas-abc",
      installParams: { appRepository: "acme/app", backlogRepository: "acme/backlog" },
      integrations: {
        github: { id: "int-1", name: "acme-github", ready: true },
      },
      agentRewrite: {
        component: "runnerClaudeCode",
        model: "claude-sonnet-4-6",
        planningModel: "claude-opus-4-6",
        credentials: { source: "hosted" },
      },
    });
    const rewrittenAgent = canvasNodes(rewrittenPlanning).find((node) => node.id === "planner-agent-no-issue");
    expect(rewrittenAgent?.configuration?.model).toBe("claude-opus-4-6");
  });

  it("falls back to the standard model when the rewrite resolved no planning model", () => {
    const planning = materializeFactoryCanvas({
      definition: getFactoryDefinition("line-planning"),
      canvasName: "Plan",
      canvasId: "canvas-abc",
      installParams: { appRepository: "acme/app", backlogRepository: "acme/backlog" },
      integrations: { github: { id: "int-1", name: "acme-github", ready: true } },
      agentRewrite: {
        component: "runnerCodex",
        model: "gpt-5",
        credentials: { source: "hosted" },
      },
    });
    const agent = canvasNodes(planning).find((node) => node.id === "planner-agent-no-issue");

    expect(agent?.component).toBe("runnerCodex");
    expect(agent?.configuration?.model).toBe("gpt-5");
  });

  it("names a model on every agent node, with or without an onboarding rewrite", () => {
    for (const factoryId of ["line-planning", "line-implementation"]) {
      const agentNodes = canvasNodes(materializeOnboardingApp(factoryId)).filter((node) =>
        AGENT_COMPONENTS.includes(node.component ?? ""),
      );

      expect(agentNodes.length).toBeGreaterThanOrEqual(1);
      for (const node of agentNodes) {
        expect(node.configuration?.model, `${factoryId}/${node.id}`).toBeTruthy();
      }
    }
  });

  it("uses the agent provider and model selected during onboarding", () => {
    for (const factoryId of ["line-planning", "line-implementation"]) {
      const canvasYaml = materializeFactoryCanvas({
        definition: getFactoryDefinition(factoryId),
        canvasName: factoryId,
        canvasId: "canvas-abc",
        installParams: { appRepository: "acme/app", backlogRepository: "acme/backlog" },
        integrations: {
          github: { id: "int-1", name: "acme-github", ready: true },
        },
        agentRewrite: {
          component: "runnerOpenRouter",
          model: "openai/gpt-4.1",
          credentials: { source: "hosted" },
        },
      });
      const agentNodes = canvasNodes(canvasYaml).filter((node) => node.component === "runnerOpenRouter");

      expect(agentNodes.length).toBeGreaterThanOrEqual(1);
      expect(agentNodes.every((node) => node.configuration?.model === "openai/gpt-4.1")).toBe(true);
    }
  });

  it("routes code work to the app repository", () => {
    const planning = materializeOnboardingApp("line-planning");
    const planningNodeIds = canvasNodes(planning).map((node) => node.id);
    expect(planning).toContain("acme/app");
    expect(planningNodeIds).toEqual(["onrun-create-plan", "planner-agent-no-issue", "add-plan-artifact"]);
    expect(planning).toMatch(/sourceId: onrun-create-plan\n\s+targetId: planner-agent-no-issue/);

    const implementation = materializeOnboardingApp("line-implementation");
    const implementationNodeIds = canvasNodes(implementation).map((node) => node.id);
    expect(implementation).toContain("acme/app");
    expect(implementation).toMatch(
      /id: add-branch-artifact[\s\S]*component: addWorkOrderArtifact[\s\S]*repository: acme\/app/,
    );
    expect(implementationNodeIds).toEqual([
      "onrun-implement",
      "create-branch",
      "add-branch-artifact",
      "implementation-agent-no-issue",
      "generate-pr-text",
      "create-draft-pr",
      "add-pr-label",
      "attach-pr-artifact",
      "set-pr-closure-note",
      "add-run-error",
    ]);
    expect(implementation).toMatch(/sourceId: add-branch-artifact\n\s+targetId: implementation-agent-no-issue/);
    expect(implementation).toMatch(/sourceId: implementation-agent-no-issue\n\s+targetId: generate-pr-text/);
    expect(implementation).toMatch(/sourceId: generate-pr-text\n\s+targetId: create-draft-pr/);
    expect(implementation).toMatch(/component: github\.createPullRequest[\s\S]*repository: acme\/app/);
    expect(implementation).toMatch(/sourceId: create-draft-pr\n\s+targetId: attach-pr-artifact/);
    expect(implementation).toMatch(/sourceId: attach-pr-artifact\n\s+targetId: add-pr-label/);
    expect(implementation).toMatch(/id: attach-pr-artifact[\s\S]*component: addPullRequest/);
    expect(implementation).toMatch(
      /id: attach-pr-artifact[\s\S]*url: '\{\{ \$\["Create Draft Pull Request"\]\.data\._links\.html\.href \}\}'/,
    );
    expect(implementation).toMatch(/id: attach-pr-artifact[\s\S]*state: open/);
    expect(Object.fromEntries(canvasNodes(implementation).map((node) => [node.id, node.concurrency?.max]))).toEqual({
      "onrun-implement": undefined,
      "create-branch": 5,
      "add-branch-artifact": 100,
      "implementation-agent-no-issue": 5,
      "generate-pr-text": 5,
      "create-draft-pr": 100,
      "add-pr-label": undefined,
      "attach-pr-artifact": 100,
      "set-pr-closure-note": undefined,
      "add-run-error": undefined,
    });
  });

  it("writes the pull request title and description before it opens the draft", () => {
    const implementation = materializeOnboardingApp("line-implementation");

    expect(implementation).toContain("id: generate-pr-text");
    expect(implementation).toContain("Missing title and/or description at /tmp/TITLE and /tmp/DESCRIPTION.md");
    expect(implementation).toMatch(
      /component: github\.createPullRequest[\s\S]*fromBase64\(previous\(\)\.data\.result\.title\)/,
    );
    expect(implementation).toMatch(
      /component: github\.createPullRequest[\s\S]*\[Work Order\]\(\{\{ order\(\)\.url \}\}\)/,
    );
    expect(implementation).toContain("Created via [SuperPlane](https://superplane.com)");
  });

  it("announces the PR-merge wait after the work order attaches the pull request", () => {
    const implementation = materializeOnboardingApp("line-implementation");

    expect(implementation).toMatch(/sourceId: add-pr-label[\s\S]*targetId: set-pr-closure-note/);
    expect(implementation).toContain("component: setWorkOrderStatusNote");
    expect(implementation).toContain("noteKey: pr-closure");
    expect(implementation).toContain("headline: Waiting for user review");
    expect(implementation).toContain(
      "The pull request is open and waiting for user review. Mention @superplaneagent in a pull request comment or review to request changes.",
    );
    expect(implementation).toContain("ctaUrl: '{{ $[\"Create Draft Pull Request\"].data.html_url }}'");
    expect(implementation).toContain("showOnlyWhenWaiting: true");
  });

  it("fails planning when the agent does not write the plan file", () => {
    const planning = materializeOnboardingApp("line-planning");
    expect(planning).toContain("No plan produced at /tmp/plan.md");
    expect(planning).toContain("exit 1");
  });

  it("fails implementation when the agent pushes no file commits", () => {
    const implementation = materializeOnboardingApp("line-implementation");
    const nodes = canvasNodes(implementation);

    const steps = nodeStepsByName(nodes, "implementation-agent-no-issue");
    const checkout = steps["Checkout Branch"]?.command ?? "";
    const commit = steps["Commit and Push"]?.command ?? "";

    expect(checkout).toContain("set -euo pipefail");
    expect(commit).toContain("No file changes and no unpushed commits");
    expect(commit).toContain("exit 1");
    expect(commit).not.toContain("already up to date on origin");
    expect(commit).toContain("git status");
    expect(commit).toContain("git log --oneline -5");
  });

  it("runs implementation prompt and commit steps in the cloned repo", () => {
    const implementation = materializeOnboardingApp("line-implementation");
    const nodes = canvasNodes(implementation);

    const steps = nodeStepsByName(nodes, "implementation-agent-no-issue");
    expect(steps["Set Up DCO Signing"]?.workingDirectory).toBe("repo");
    expect(steps["Implementation"]?.workingDirectory).toBe("repo");
    expect(steps["Commit and Push"]?.workingDirectory).toBe("repo");
  });
});

describe("setup factory event apps", () => {
  // Issue intake is a factory intake now, so setup provisions it through the
  // intake API rather than as a bundled app.
  it("provisions PR closure outside the factory line", () => {
    expect(ONBOARDING_EVENT_APPS).toEqual(["pr-closure"]);
    expect(ONBOARDING_LINE_APPS.map((app) => app.factoryId)).not.toContain("pr-closure");
    expect(ONBOARDING_LINE_APPS.map((app) => app.factoryId)).not.toContain("line-pr");
  });

  it("closes the work order when a factory pull request is closed", () => {
    const canvasYaml = materializeOnboardingApp("pr-closure");

    expect(canvasYaml).toMatch(/component: github\.onPullRequest[\s\S]*actions:[\s\S]*- closed/);
    expect(canvasYaml).toMatch(/component: github\.onPullRequest[\s\S]*repository: acme\/app/);
    expect(canvasYaml).toContain("component: findPullRequest");
    expect(canvasYaml).toContain("component: addPullRequestActivity");
    expect(canvasYaml).toContain("component: updatePullRequest");
    expect(canvasYaml).toContain("mergedAt: '{{ root().data.pull_request.merged_at }}'");
    expect(canvasYaml).toContain("closedAt: '{{ root().data.pull_request.closed_at }}'");
    expect(canvasYaml).toContain("result: completed");
    expect(canvasYaml).toContain("result: rejected");
    expect(canvasYaml).toContain("id: int-1");
    expect(canvasYaml).toContain("name: acme-github");
    expect(canvasYaml).not.toContain("{{ install_params.");
    expect(canvasYaml).not.toContain(FACTORY_CANVAS_ID_PLACEHOLDER);
    expect(canvasYaml).not.toContain("superplanehq");
    expect(canvasYaml).not.toMatch(/component: onRun/);
  });
});
