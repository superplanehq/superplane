import { describe, expect, it } from "vitest";
import yaml from "js-yaml";

import { getFactoryDefinition, ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS } from "./index";
import { FACTORY_CANVAS_ID_PLACEHOLDER, materializeFactoryCanvas } from "./materializeFactoryTemplate";

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

  it("keeps the planning agent on Opus when the coding agent stays Claude Code", () => {
    const planning = materializeOnboardingApp("line-planning");
    const implementation = materializeOnboardingApp("line-implementation");
    const planningAgent = canvasNodes(planning).find((node) => node.id === "planner-agent-no-issue");
    const implementationAgent = canvasNodes(implementation).find((node) => node.id === "implementation-agent-no-issue");

    expect(planningAgent?.configuration?.model).toBe("opus");
    expect(implementationAgent?.configuration?.model).toBe("sonnet");

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
        model: "sonnet",
        credentials: { source: "hosted" },
      },
    });
    const rewrittenAgent = canvasNodes(rewrittenPlanning).find((node) => node.id === "planner-agent-no-issue");
    expect(rewrittenAgent?.configuration?.model).toBe("opus");
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

      expect(agentNodes).toHaveLength(1);
      expect(agentNodes[0]?.configuration?.model).toBe("openai/gpt-4.1");
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
      "create-draft-pr",
      "attach-pr-artifact",
    ]);
    expect(implementation).toMatch(/sourceId: add-branch-artifact\n\s+targetId: implementation-agent-no-issue/);
    expect(implementation).toMatch(/sourceId: implementation-agent-no-issue\n\s+targetId: create-draft-pr/);
    expect(implementation).toMatch(/component: github\.createPullRequest[\s\S]*repository: acme\/app/);
    expect(implementation).toMatch(/sourceId: create-draft-pr\n\s+targetId: attach-pr-artifact/);
    expect(implementation).toMatch(
      /id: attach-pr-artifact[\s\S]*artifactType: pr[\s\S]*state: open[\s\S]*url: '\{\{ \$\["Create Draft Pull Request"\]\.data\._links\.html\.href \}\}'/,
    );
    expect(Object.fromEntries(canvasNodes(implementation).map((node) => [node.id, node.concurrency?.max]))).toEqual({
      "onrun-implement": undefined,
      "create-branch": 5,
      "add-branch-artifact": 100,
      "implementation-agent-no-issue": 5,
      "create-draft-pr": 100,
      "attach-pr-artifact": 100,
    });

    const pr = materializeOnboardingApp("line-pr");
    expect(pr).toMatch(/component: github\.createPullRequest[\s\S]*repository: acme\/app/);
  });

  it("links the pull request back to the work order", () => {
    const pr = materializeOnboardingApp("line-pr");

    expect(pr).toMatch(/component: github\.createPullRequest[\s\S]*\[Work Order\]\(\{\{ order\(\)\.url \}\}\)/);
    expect(pr).toContain("Created via [SuperPlane](https://superplane.com)");
  });

  it("announces the PR-merge wait after the work order attaches the pull request", () => {
    const pr = materializeOnboardingApp("line-pr");

    expect(pr).toMatch(/sourceId: attach-pr-artifact[\s\S]*targetId: set-pr-closure-note/);
    expect(pr).toContain("component: setWorkOrderStatusNote");
    expect(pr).toContain("noteKey: pr-closure");
    expect(pr).toContain("headline: Listening for user review");
    expect(pr).toContain("ctaUrl: '{{ $[\"Create Draft Pull Request\"].data.html_url }}'");
    expect(pr).toContain("showOnlyWhenWaiting: true");
  });

  it("fails planning when the agent does not write the plan file", () => {
    const planning = materializeOnboardingApp("line-planning");
    expect(planning).toContain("No plan produced at /tmp/plan.md");
    expect(planning).toContain("exit 1");
  });

  it("fails PR title generation when the agent does not write title files", () => {
    const pr = materializeOnboardingApp("line-pr");
    expect(pr).toContain("Missing title and/or description at /tmp/TITLE and /tmp/DESCRIPTION.md");
    expect(pr).toContain("exit 1");
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
  });

  it("closes the work order when a factory pull request is closed", () => {
    const canvasYaml = materializeOnboardingApp("pr-closure");

    expect(canvasYaml).toMatch(/component: github\.onPullRequest[\s\S]*actions:[\s\S]*- closed/);
    expect(canvasYaml).toMatch(/component: github\.onPullRequest[\s\S]*repository: acme\/app/);
    expect(canvasYaml).toContain("component: findWorkOrder");
    expect(canvasYaml).toContain("by: artifactKey");
    expect(canvasYaml).toContain("component: updateWorkOrderArtifact");
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
