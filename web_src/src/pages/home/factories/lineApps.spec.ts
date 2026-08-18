import { describe, expect, it } from "vitest";

import { getFactoryDefinition, ONBOARDING_LINE_APPS } from "./index";
import { FACTORY_CANVAS_ID_PLACEHOLDER, materializeFactoryCanvas } from "./materializeFactoryTemplate";

function materializeLineApp(factoryId: string) {
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

describe("onboarding factory line apps", () => {
  it("orders the apps as plan, implement, open PR", () => {
    expect(ONBOARDING_LINE_APPS.map((app) => app.factoryId)).toEqual([
      "line-planning",
      "line-implementation",
      "line-pr",
    ]);
    expect(ONBOARDING_LINE_APPS.map((app) => app.lineStepName)).toEqual([
      "Create Implementation Plan",
      "Implement",
      "Open Pull Request",
    ]);
  });

  it("exposes a single onRun entrypoint per app that the line calls", () => {
    for (const app of ONBOARDING_LINE_APPS) {
      const canvasYaml = materializeLineApp(app.factoryId);
      expect(canvasYaml).toMatch(new RegExp(`id: ${app.entrypointNodeId}[\\s\\S]*component: onRun`));
    }
  });

  it("materializes repositories and integration wiring, leaving no template placeholders", () => {
    for (const app of ONBOARDING_LINE_APPS) {
      const canvasYaml = materializeLineApp(app.factoryId);
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
    const planning = materializeLineApp("line-planning");
    expect(planning).toContain("acme/app");
    expect(planning).toContain("acme/backlog");

    const implementation = materializeLineApp("line-implementation");
    expect(implementation).toContain("acme/app");

    const pr = materializeLineApp("line-pr");
    expect(pr).toMatch(/component: github\.createPullRequest[\s\S]*repository: acme\/app/);
  });
});
