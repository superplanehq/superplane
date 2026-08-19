import { describe, expect, it } from "vitest";

import { getFactoryDefinition } from "./index";
import {
  FACTORY_CANVAS_ID_PLACEHOLDER,
  buildFactoryRunParameters,
  materializeFactoryCanvas,
  materializeFactoryConsole,
  normalizeFactoryInstallParams,
  substituteInstallParams,
  wireFactoryIntegrations,
} from "./materializeFactoryTemplate";

function materializeSoftwareFactory(installParams: Record<string, string> = { repository: "acme/web" }) {
  return materializeFactoryCanvas({
    definition: getFactoryDefinition("software-factory"),
    canvasName: "My Factory",
    canvasId: "canvas-123",
    installParams,
    integrations: {
      github: { id: "int-1", name: "acme-github", ready: true },
      claude: { id: "int-2", name: "acme-claude", ready: true },
    },
  });
}

describe("materializeFactoryTemplate", () => {
  it("substitutes install_params placeholders", () => {
    const yaml = 'repository: "{{ install_params.appRepository }}"';
    expect(
      substituteInstallParams(yaml, {
        appRepository: "acme/web",
      }),
    ).toBe('repository: "acme/web"');
  });

  it("leaves unknown install_params placeholders unresolved", () => {
    expect(substituteInstallParams("x: {{ install_params.missing }}", {})).toBe("x: {{ install_params.missing }}");
  });

  it("maps legacy repository onto app and backlog repositories", () => {
    expect(normalizeFactoryInstallParams({ repository: "acme/web" })).toEqual({
      repository: "acme/web",
      appRepository: "acme/web",
      backlogRepository: "acme/web",
    });
    expect(
      normalizeFactoryInstallParams({
        repository: "acme/web",
        appRepository: "acme/app",
        backlogRepository: "acme/backlog",
      }),
    ).toEqual({
      repository: "acme/web",
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
    });
  });

  it("wires integration refs onto matching components", () => {
    const wired = wireFactoryIntegrations(
      `
apiVersion: v1
kind: Canvas
spec:
  nodes:
    - id: create-issue
      component: github.createIssue
    - id: draft-issue
      component: runnerClaudeCode
      configuration:
        credentials:
          source: integration
          integration:
            name: claude
`,
      { "github.createIssue": "github" },
      {
        github: { id: "int-1", name: "acme-github", ready: true },
        claude: { id: "int-2", name: "acme-claude", ready: true },
      },
    );

    expect(wired).toContain("id: int-1");
    expect(wired).toContain("name: acme-github");
    expect(wired).toContain("name: acme-claude");
    expect(wired).toContain("runnerClaudeCode");
    expect(wired).toMatch(/credentials:[\s\S]*name: acme-claude/);
    expect(wired).not.toMatch(/credentials:[\s\S]*name: claude\b/);
  });

  it("materializes factory identity, repo, and github wiring", () => {
    const canvasYaml = materializeSoftwareFactory();
    const definition = getFactoryDefinition("software-factory");

    expect(canvasYaml).toContain("name: My Factory");
    expect(canvasYaml).toContain("acme/web");
    expect(canvasYaml).toContain("create-task-start");
    expect(canvasYaml).toContain("work-order-dispatch");
    expect(canvasYaml).toContain("id: int-1");
    expect(canvasYaml).toContain("app: canvas-123");
    expect(canvasYaml).not.toContain(FACTORY_CANVAS_ID_PLACEHOLDER);
    expect(canvasYaml).not.toContain("{{ install_params.");
    expect(definition.integrations).toEqual(["github", "claude"]);
  });

  it("routes issue work to backlog and branch/PR/CI work to app repositories", () => {
    const canvasYaml = materializeSoftwareFactory({
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
    });

    expect(canvasYaml).toContain("acme/app");
    expect(canvasYaml).toContain("acme/backlog");
    expect(canvasYaml).toContain("fixes acme/backlog#");
    expect(canvasYaml).toMatch(/component: github\.onIssue[\s\S]*repository: acme\/backlog/);
    expect(canvasYaml).toMatch(/component: github\.createIssue[\s\S]*repository: acme\/backlog/);
    expect(canvasYaml).toMatch(/component: github\.createPullRequest[\s\S]*repository: acme\/app/);
    expect(canvasYaml).toMatch(/component: github\.onPullRequest[\s\S]*repository: acme\/app/);
    expect(canvasYaml).toMatch(/id: runner-create-branch-2[\s\S]*name: REPO[\s\S]*value: acme\/app/);
    expect(canvasYaml).toMatch(/id: analyze-create-issue[\s\S]*name: REPO[\s\S]*value: acme\/app/);
    expect(canvasYaml).not.toContain("{{ install_params.");
    expect(canvasYaml).not.toContain("install_params.repository");
  });

  it("connects work-order dispatch onRun to the create-task pipeline", () => {
    const canvasYaml = materializeSoftwareFactory();

    expect(canvasYaml).toMatch(/id: work-order-dispatch[\s\S]*component: onRun/);
    expect(canvasYaml).toContain("sourceId: work-order-dispatch");
    expect(canvasYaml).toContain("targetId: analyze-create-issue");
    expect(canvasYaml).toContain("targetId: submit-task-memory");
    expect(canvasYaml).toContain("order() != nil");
    expect(canvasYaml).toContain("order().title");
    expect(canvasYaml).toContain("order().description");
    expect(canvasYaml).toContain("root().data.description");
  });

  it("materializes runners with integration-sourced credentials", () => {
    const canvasYaml = materializeSoftwareFactory();

    expect(canvasYaml).toContain("runnerClaudeCode");
    expect(canvasYaml).toContain("@superplaneagent");
    expect(canvasYaml).toContain('name == "factory"');
    expect(canvasYaml).toContain("REPLACE ME WITH");
    expect(canvasYaml).toContain("source: integration");
    expect(canvasYaml).toContain("environmentFrom:");
    expect(canvasYaml).toContain("name: acme-claude");
    expect(canvasYaml).toContain("name: acme-github");
    expect(canvasYaml).not.toContain("anthropicApiKey");
    expect(canvasYaml).not.toContain("install_params.anthropic_api_key");
    expect(canvasYaml).not.toContain("install_params.github_token");
    expect(canvasYaml).not.toContain("REPLACE_ME_WITH_");
  });

  it("materializes console canvasId to the installed canvas", () => {
    const definition = getFactoryDefinition("software-factory");
    const consoleYaml = materializeFactoryConsole(definition, "My Factory", "canvas-123");
    expect(consoleYaml).toContain("name: My Factory");
    expect(consoleYaml).toContain("canvasId: canvas-123");
    expect(consoleYaml).toContain("@superplaneagent");
    expect(consoleYaml).not.toContain(FACTORY_CANVAS_ID_PLACEHOLDER);
    expect(consoleYaml).not.toContain("REPLACE_ME_WITH_");
  });

  it("builds invoke parameters from the starting task prompt", () => {
    const definition = getFactoryDefinition("software-factory");
    expect(buildFactoryRunParameters(definition, "fix a bug")).toEqual({
      template: "Create Task",
      description: "fix a bug",
    });
  });

  it("exposes separate app and backlog install params on the bundled definition", () => {
    const definition = getFactoryDefinition("software-factory");
    expect(definition.installParams.map((param) => param.name)).toEqual(["appRepository", "backlogRepository"]);
  });
});
