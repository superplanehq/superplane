import { describe, expect, it } from "vitest";
import yaml from "js-yaml";

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

  it("fails software-factory implementation when the agent pushes no file commits", () => {
    const canvasYaml = materializeSoftwareFactory();
    const steps = nodeStepsByName(canvasNodes(canvasYaml), "run-claude-code-implementation");
    const checkout = steps["Checkout branch"]?.command ?? "";
    const commit = steps["Commit and push"]?.command ?? "";

    expect(checkout).toContain("set -euo pipefail");
    expect(commit).toContain("No file changes and no unpushed commits");
    expect(commit).toContain("exit 1");
    expect(commit).not.toContain("already up to date on origin");
    expect(commit).toContain("git status");
    expect(commit).toContain("git log --oneline -5");
  });

  it("runs software-factory implementation prompt and commit steps in the cloned repo", () => {
    const canvasYaml = materializeSoftwareFactory();
    const steps = nodeStepsByName(canvasNodes(canvasYaml), "run-claude-code-implementation");

    expect(steps["Set Up DCO Signing"]?.workingDirectory).toBe("repo");
    expect(steps["Implementation"]?.workingDirectory).toBe("repo");
    expect(steps["Commit and push"]?.workingDirectory).toBe("repo");
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

  it("keeps the planning agent on Opus when Claude Code credentials become hosted", () => {
    const canvasYaml = materializeFactoryCanvas({
      definition: getFactoryDefinition("line-planning"),
      canvasName: "Plan",
      canvasId: "canvas-hosted-plan",
      installParams: { appRepository: "acme/app", backlogRepository: "acme/backlog" },
      integrations: {
        github: { id: "int-1", name: "acme-github", ready: true },
      },
      agentRewrite: {
        component: "runnerClaudeCode",
        model: "claude-sonnet-4-6",
        credentials: { source: "hosted" },
      },
    });

    expect(canvasYaml).toMatch(/id: planner-agent-no-issue[\s\S]*model: opus/);
    expect(canvasYaml).not.toContain("model: claude-sonnet-4-6");
  });

  it("rewrites Claude Code credentials to hosted when Claude is not connected", () => {
    const canvasYaml = materializeFactoryCanvas({
      definition: getFactoryDefinition("line-implementation"),
      canvasName: "Implementation",
      canvasId: "canvas-hosted",
      installParams: { appRepository: "acme/app", backlogRepository: "acme/backlog" },
      integrations: {
        github: { id: "int-1", name: "acme-github", ready: true },
      },
      agentRewrite: {
        component: "runnerClaudeCode",
        model: "claude-sonnet-4-6",
        credentials: { source: "hosted" },
      },
    });

    expect(canvasYaml).toMatch(/component: runnerClaudeCode[\s\S]*credentials:[\s\S]*source: hosted/);
    expect(canvasYaml).toContain("model: claude-sonnet-4-6");
    expect(canvasYaml).not.toMatch(/credentials:[\s\S]*source: integration[\s\S]*name: claude/);
    expect(canvasYaml).not.toContain("model: sonnet");
  });

  it("rewrites Claude Code nodes to hosted OpenRouter with an allowlisted model", () => {
    const canvasYaml = materializeFactoryCanvas({
      definition: getFactoryDefinition("line-planning"),
      canvasName: "Planning",
      canvasId: "canvas-openrouter",
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

    expect(canvasYaml).toMatch(/component: runnerOpenRouter[\s\S]*credentials:[\s\S]*source: hosted/);
    expect(canvasYaml).toContain("model: openai/gpt-4.1");
    expect(canvasYaml).not.toContain("runnerClaudeCode");
  });

  it("rewrites Claude Code nodes to an OpenRouter integration", () => {
    const canvasYaml = materializeFactoryCanvas({
      definition: getFactoryDefinition("line-pr"),
      canvasName: "Verify",
      canvasId: "canvas-or-byok",
      installParams: { appRepository: "acme/app", backlogRepository: "acme/backlog" },
      integrations: {
        github: { id: "int-1", name: "acme-github", ready: true },
        openrouter: { id: "int-3", name: "acme-openrouter", ready: true },
      },
      agentRewrite: {
        component: "runnerOpenRouter",
        model: "anthropic/claude-sonnet-4-6",
        credentials: { source: "integration", name: "acme-openrouter" },
      },
    });

    expect(canvasYaml).toMatch(/component: runnerOpenRouter[\s\S]*credentials:[\s\S]*name: acme-openrouter/);
    expect(canvasYaml).not.toContain("runnerClaudeCode");
  });
});
