import { describe, expect, it } from "vitest";
import yaml from "js-yaml";

import { getFactoryDefinition, ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS } from "./index";
import { FACTORY_CANVAS_ID_PLACEHOLDER, materializeFactoryCanvas } from "./materializeFactoryTemplate";

type AgentStep = { name?: string; command?: string; workingDirectory?: string };

type CanvasNode = {
  id?: string;
  configuration?: { steps?: AgentStep[] };
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
  it("orders the apps as plan, implement, open PR", () => {
    expect(ONBOARDING_LINE_APPS.map((app) => app.factoryId)).toEqual([
      "line-planning",
      "line-implementation",
      "line-pr",
    ]);
  });

  it("exposes a single onRun entrypoint per app that the line calls", () => {
    for (const app of ONBOARDING_LINE_APPS) {
      const canvasYaml = materializeOnboardingApp(app.factoryId);
      expect(canvasYaml).toMatch(new RegExp(`id: ${app.entrypointNodeId}[\\s\\S]*component: onRun`));
    }
  });

  it("materializes repositories and integration wiring, leaving no template placeholders", () => {
    for (const app of ONBOARDING_LINE_APPS) {
      const canvasYaml = materializeOnboardingApp(app.factoryId);
      expect(canvasYaml).toContain("runnerOpenRouter");
      expect(canvasYaml).toContain("source: hosted");
      expect(canvasYaml).toContain("name: acme-github");
      expect(canvasYaml).not.toContain("name: acme-claude");
      expect(canvasYaml).not.toContain("{{ install_params.");
      expect(canvasYaml).not.toContain(FACTORY_CANVAS_ID_PLACEHOLDER);
      // No org-specific bindings copied from the production canvases.
      expect(canvasYaml).not.toContain("superplanehq");
      expect(canvasYaml).not.toContain("claude-superplane-apps");
    }
  });

  it("routes code work to the app repository", () => {
    const planning = materializeOnboardingApp("line-planning");
    expect(planning).toContain("acme/app");
    expect(planning).toContain("acme/backlog");

    const implementation = materializeOnboardingApp("line-implementation");
    expect(implementation).toContain("acme/app");
    expect(implementation).toMatch(
      /id: add-branch-artifact[\s\S]*component: addWorkOrderArtifact[\s\S]*repository: acme\/app/,
    );
    expect(implementation).toMatch(/sourceId: implementation-agent\n\s+targetId: create-draft-pr/);
    expect(implementation).toMatch(/component: github\.createPullRequest[\s\S]*repository: acme\/app/);
    expect(implementation).toMatch(/sourceId: create-draft-pr\n\s+targetId: attach-pr-artifact/);
    expect(implementation).toMatch(/id: attach-pr-artifact[\s\S]*artifactType: pr/);

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

    for (const nodeId of ["implementation-agent", "implementation-agent-no-issue"]) {
      const steps = nodeStepsByName(nodes, nodeId);
      const checkout = steps["Checkout Branch"]?.command ?? "";
      const commit = steps["Commit and Push"]?.command ?? "";

      expect(checkout).toContain("set -euo pipefail");
      expect(commit).toContain("No file changes and no unpushed commits");
      expect(commit).toContain("exit 1");
      expect(commit).not.toContain("already up to date on origin");
      expect(commit).toContain("git status");
      expect(commit).toContain("git log --oneline -5");
    }
  });

  it("runs implementation prompt and commit steps in the cloned repo", () => {
    const implementation = materializeOnboardingApp("line-implementation");
    const nodes = canvasNodes(implementation);

    for (const nodeId of ["implementation-agent", "implementation-agent-no-issue"]) {
      const steps = nodeStepsByName(nodes, nodeId);
      expect(steps["Set Up DCO Signing"]?.workingDirectory).toBe("repo");
      expect(steps["Implementation"]?.workingDirectory).toBe("repo");
      expect(steps["Commit and Push"]?.workingDirectory).toBe("repo");
    }
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
