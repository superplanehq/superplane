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

type FactoryCanvasNode = {
  id?: string;
  configuration?: { script?: unknown };
};

// Materializes the Software Factory and returns the generated Create Branch
// runner script, so regression coverage can assert on its actual behavior
// instead of substring-matching the whole ~1,400-line template.
function findCreateBranchScript(canvasYaml: string): string {
  const doc = yaml.load(canvasYaml) as { spec?: { nodes?: FactoryCanvasNode[] } };
  const node = doc.spec?.nodes?.find((n) => n.id === "runner-create-branch-2");
  const script = node?.configuration?.script;
  if (typeof script !== "string") {
    throw new Error("runner-create-branch-2 configuration.script not found in materialized canvas");
  }
  return script;
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

describe("Software Factory Create Branch", () => {
  it("handles empty repositories when the default branch has no remote ref", () => {
    const script = findCreateBranchScript(materializeSoftwareFactory());

    // Resolves the configured default branch dynamically; never hardcodes `main`.
    expect(script).toContain('gh api "repos/${REPO}" --jq .default_branch');
    expect(script).not.toMatch(/--branch main\b/);
    expect(script).not.toMatch(/checkout --orphan main\b/);
    expect(script).not.toMatch(/push -u origin main\b/);
    expect(script).not.toMatch(/checkout -b main\b/);

    // Lists remote branch refs once so real git failures (auth/network/permission)
    // are not misclassified as an empty repository.
    expect(script).toMatch(/git ls-remote --heads "\$GITURL"/);
    expect(script).toMatch(/Failed to list remote branches/);
    expect(script).toMatch(/exit 1/);

    // Existing-repository path is preserved: shallow clone of the default branch.
    expect(script).toMatch(/git clone --depth 1 --branch "\$BASE"/);

    // Empty-repository fallback: clone without --branch, initialize, push base.
    expect(script).toMatch(/git clone "\$GITURL" "\$WORKDIR"/);
    expect(script).toMatch(/git checkout --orphan "\$BASE"/);
    expect(script).toMatch(/git push -u origin "\$BASE"/);

    // Work branch is still created and pushed from the base commit.
    expect(script).toMatch(/git checkout -b "\$BRANCH"/);
    expect(script).toMatch(/git push -u origin "\$BRANCH"/);

    // Result JSON contract is unchanged (base = resolved default branch).
    expect(script).toContain("SUPERPLANE_RESULT_FILE");
    expect(script).toMatch(/"branch":"%s"/);
    expect(script).toMatch(/"base":"%s"/);
  });
});
