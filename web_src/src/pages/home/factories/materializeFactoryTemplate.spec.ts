import { describe, expect, it } from "vitest";
import yaml from "js-yaml";

import { getFactoryDefinition } from "./index";
import {
  FACTORY_CANVAS_ID_PLACEHOLDER,
  buildFactoryRunParameters,
  materializeFactoryCanvas,
  materializeFactoryConsole,
  substituteInstallParams,
  wireFactoryIntegrations,
} from "./materializeFactoryTemplate";

function materializeSoftwareFactory() {
  return materializeFactoryCanvas({
    definition: getFactoryDefinition("software-factory"),
    canvasName: "My Factory",
    canvasId: "canvas-123",
    installParams: {
      repository: "acme/web",
    },
    integrations: {
      github: { id: "int-1", name: "acme-github", ready: true },
      claude: { id: "int-2", name: "acme-claude", ready: true },
    },
  });
}

describe("materializeFactoryTemplate", () => {
  it("substitutes install_params placeholders", () => {
    const yaml = 'repository: "{{ install_params.repository }}"';
    expect(
      substituteInstallParams(yaml, {
        repository: "acme/web",
      }),
    ).toBe('repository: "acme/web"');
  });

  it("leaves unknown install_params placeholders unresolved", () => {
    expect(substituteInstallParams("x: {{ install_params.missing }}", {})).toBe("x: {{ install_params.missing }}");
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
    expect(canvasYaml).toContain("id: int-1");
    expect(canvasYaml).toContain("app: canvas-123");
    expect(canvasYaml).not.toContain(FACTORY_CANVAS_ID_PLACEHOLDER);
    expect(canvasYaml).not.toContain("{{ install_params.");
    expect(definition.integrations).toEqual(["github", "claude"]);
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

  it("acknowledges every GitHub instruction path with an eyes reaction", () => {
    const canvas = yaml.load(materializeSoftwareFactory()) as {
      spec?: {
        nodes?: Array<{
          id?: string;
          component?: string;
          configuration?: Record<string, unknown>;
          integration?: { id?: string; name?: string };
        }>;
        edges?: Array<{ sourceId?: string; targetId?: string; channel?: string }>;
      };
    };
    const nodesById = new Map((canvas.spec?.nodes ?? []).map((node) => [node.id, node]));
    const edges = canvas.spec?.edges ?? [];

    expect(nodesById.get("acknowledge-issue-instruction")).toMatchObject({
      component: "github.addIssueReaction",
      configuration: {
        repository: "acme/web",
        issueNumber: "{{ root().data.issue.number }}",
        content: "eyes",
      },
      integration: { id: "int-1", name: "acme-github" },
    });
    expect(nodesById.get("acknowledge-comment-instruction")).toMatchObject({
      component: "github.addReaction",
      configuration: {
        repository: "acme/web",
        target: "issueComment",
        commentId: "{{ root().data.comment.id }}",
        content: "eyes",
      },
      integration: { id: "int-1", name: "acme-github" },
    });
    expect(nodesById.get("acknowledge-pr-review-comment-instruction")).toMatchObject({
      component: "github.addReaction",
      configuration: {
        repository: "acme/web",
        target: "reviewComment",
        commentId: "{{ root().data.comment.id }}",
        content: "eyes",
      },
      integration: { id: "int-1", name: "acme-github" },
    });

    expect(nodesById.get("assigned-to-agent-filter")).toMatchObject({
      configuration: {
        expression: expect.stringContaining("root().data.issue.pull_request == nil"),
      },
    });
    expect(nodesById.get("factory-pr-review-comment-filter")).toMatchObject({
      configuration: {
        expression: expect.stringContaining("root().data.comment != nil"),
      },
    });

    expect(edges).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          sourceId: "assigned-to-agent-filter",
          targetId: "acknowledge-issue-instruction",
        }),
        expect.objectContaining({
          sourceId: "label-is-factory-filter",
          targetId: "acknowledge-issue-instruction",
        }),
        expect.objectContaining({
          sourceId: "component-node-4m9qti",
          targetId: "acknowledge-comment-instruction",
        }),
        expect.objectContaining({
          sourceId: "factory-pr-comment-filter",
          targetId: "acknowledge-comment-instruction",
        }),
        expect.objectContaining({
          sourceId: "factory-pr-comment-filter",
          targetId: "apply-user-request-claude",
        }),
        expect.objectContaining({
          sourceId: "factory-pr-review-comment-filter",
          targetId: "acknowledge-pr-review-comment-instruction",
        }),
        expect.objectContaining({
          sourceId: "factory-pr-review-comment-filter",
          targetId: "apply-user-request-claude",
        }),
      ]),
    );

    expect(edges).not.toContainEqual(
      expect.objectContaining({
        sourceId: "on-pr-comment-trigger",
        targetId: "apply-user-request-claude",
      }),
    );
    expect(edges).not.toContainEqual(
      expect.objectContaining({
        sourceId: "on-pr-review-comment-trigger",
        targetId: "apply-user-request-claude",
      }),
    );
    expect(edges).not.toContainEqual(
      expect.objectContaining({
        sourceId: "on-pr-comment-trigger",
        targetId: "acknowledge-comment-instruction",
      }),
    );
    expect(edges).not.toContainEqual(
      expect.objectContaining({
        sourceId: "on-pr-review-comment-trigger",
        targetId: "acknowledge-pr-review-comment-instruction",
      }),
    );
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
});
