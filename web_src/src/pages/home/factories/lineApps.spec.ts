import { describe, expect, it } from "vitest";

import { getFactoryDefinition, ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS } from "./index";
import { FACTORY_CANVAS_ID_PLACEHOLDER, materializeFactoryCanvas } from "./materializeFactoryTemplate";

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
      expect(canvasYaml).toContain("name: acme-claude");
      expect(canvasYaml).toContain("name: acme-github");
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
    expect(pr).toContain("headline: Review the pull request");
    expect(pr).toContain("ctaUrl: '{{ $[\"Create Draft Pull Request\"].data.html_url }}'");
    expect(pr).toContain("showOnlyWhenWaiting: true");
  });
});

describe("setup factory event apps", () => {
  it("provisions issue intake and PR closure outside the factory line", () => {
    expect(ONBOARDING_EVENT_APPS).toEqual(["issue-intake", "pr-closure"]);
    expect(ONBOARDING_LINE_APPS.map((app) => app.factoryId)).not.toContain("issue-intake");
    expect(ONBOARDING_LINE_APPS.map((app) => app.factoryId)).not.toContain("pr-closure");
  });

  it("deduplicates source issues before creating a work order", () => {
    const canvasYaml = materializeOnboardingApp("issue-intake");

    expect(canvasYaml).toMatch(/component: github\.onIssue[\s\S]*actions:[\s\S]*- labeled/);
    expect(canvasYaml).toMatch(/component: github\.onIssue[\s\S]*actions:[\s\S]*- assigned/);
    expect(canvasYaml).toMatch(/component: github\.onIssue[\s\S]*repository: acme\/backlog/);
    expect(canvasYaml).toContain('root().data.label.name == "factory"');
    expect(canvasYaml).toContain('root().data.assignee.login == "superplaneagent"');
    expect(canvasYaml).toContain("component: findWorkOrder");
    expect(canvasYaml).toContain("by: artifactKey");
    expect(canvasYaml).toContain("artifactKey: '{{ root().data.issue.html_url }}'");
    expect(canvasYaml).toMatch(/channel: notFound[\s\S]*sourceId: find-existing-work-order[\s\S]*targetId: create-work-order/);
    expect(canvasYaml).toMatch(/component: createWorkOrder[\s\S]*title: '{{ root\(\)\.data\.issue\.title }}'/);
    expect(canvasYaml).toContain("description: '{{ root().data.issue.body }}'");
    expect(canvasYaml).toContain("component: addWorkOrderArtifact");
    expect(canvasYaml).toContain("artifactType: link");
    expect(canvasYaml).toContain("orderId: '{{ $[\"Create Work Order\"].data.workOrder.id }}'");
    expect(canvasYaml).toContain("url: '{{ root().data.issue.html_url }}'");
    expect(canvasYaml).toContain("id: int-1");
    expect(canvasYaml).toContain("name: acme-github");
    expect(canvasYaml).not.toContain("{{ install_params.");
    expect(canvasYaml).not.toContain(FACTORY_CANVAS_ID_PLACEHOLDER);
    expect(canvasYaml).not.toContain("superplanehq");
    expect(canvasYaml).not.toMatch(/component: onRun/);
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
