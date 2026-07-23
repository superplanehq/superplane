import { describe, expect, it } from "vitest";

import { getFactoryDefinition } from "./index";
import {
  buildFactoryRunParameters,
  materializeFactoryCanvas,
  substituteInstallParams,
  wireFactoryIntegrations,
} from "./materializeFactoryTemplate";

describe("materializeFactoryTemplate", () => {
  it("substitutes install_params placeholders", () => {
    const yaml = 'repository: "{{ install_params.repository }}"\nsecret: "{{ install_params.anthropic_api_key }}"';
    expect(
      substituteInstallParams(yaml, {
        repository: "acme/web",
        anthropic_api_key: "anthropic-prod",
      }),
    ).toBe('repository: "acme/web"\nsecret: "anthropic-prod"');
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
    - id: run-claude-code
      component: claude.runCodeAgent
`,
      { "github.createIssue": "github", "claude.runCodeAgent": "claude" },
      {
        github: { id: "int-1", name: "acme-github" },
        claude: { id: "int-2", name: "acme-claude" },
      },
    );

    expect(wired).toContain("id: int-1");
    expect(wired).toContain("name: acme-github");
    expect(wired).toContain("id: int-2");
    expect(wired).toContain("name: acme-claude");
    expect(wired).toContain("claude.runCodeAgent");
    expect(wired.match(/integration:/g)?.length).toBe(2);
  });

  it("materializes the software factory canvas with params and integrations", () => {
    const definition = getFactoryDefinition("software-factory");
    const canvasYaml = materializeFactoryCanvas({
      definition,
      canvasName: "My Factory",
      installParams: {
        repository: "acme/web",
      },
      integrations: {
        github: { id: "int-1", name: "acme-github" },
        claude: { id: "int-2", name: "acme-claude" },
      },
    });

    expect(canvasYaml).toContain("name: My Factory");
    expect(canvasYaml).toContain("acme/web");
    expect(canvasYaml).toContain("claude.runCodeAgent");
    expect(canvasYaml).toContain("github-token");
    expect(canvasYaml).toContain("id: int-1");
    expect(canvasYaml).toContain("id: int-2");
    expect(canvasYaml).toContain('{{ parameters["prompt"] }}');
    expect(canvasYaml).toContain("{{ root().data.prompt }}");
    expect(canvasYaml).not.toContain("runnerClaudeCode");
    expect(canvasYaml).not.toContain("{{ install_params.");
  });

  it("builds invoke parameters from the starting task prompt", () => {
    const definition = getFactoryDefinition("software-factory");
    expect(buildFactoryRunParameters(definition, "fix a bug")).toEqual({
      template: "Run task",
      prompt: "fix a bug",
    });
  });
});
