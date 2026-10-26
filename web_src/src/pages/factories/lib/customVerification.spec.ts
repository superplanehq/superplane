import { describe, expect, it } from "bun:test";

import { buildColumnAutomations, columnAutomationOpenPath } from "./columnAutomations";
import { customVerificationDraft, customVerificationModeCopy, customVerificationSettings } from "./customVerification";

describe("custom verification settings", () => {
  it("maps saved agent and ordered steps into the shared editor and back", () => {
    const draft = customVerificationDraft({
      automation: { id: "verification-1", name: "Security review" },
      provider: "bitbucket",
      settings: {
        startMode: "START_MODE_AUTOMATIC",
        agent: {
          component: "runnerClaudeCode",
          model: "claude-sonnet",
          credentialSource: "integration",
          credentialIntegrationName: "claude-team",
        },
        steps: [
          { name: "Review", type: "TYPE_PROMPT", prompt: "Review this change.", workingDirectory: "repo" },
          { name: "Test", type: "TYPE_BASH", command: "make test", workingDirectory: "repo" },
        ],
      },
    });

    expect(draft.components[0]?.configuration.steps).toEqual([
      { name: "Review", type: "prompt", prompt: "Review this change.", command: undefined, workingDirectory: "repo" },
      { name: "Test", type: "bash", prompt: undefined, command: "make test", workingDirectory: "repo" },
    ]);
    expect(customVerificationSettings("automatic", draft, true)).toEqual({
      startMode: "START_MODE_AUTOMATIC",
      icon: "sparkles",
      agent: {
        component: "runnerClaudeCode",
        model: "claude-sonnet",
        credentialSource: "integration",
        credentialIntegrationName: "claude-team",
      },
      steps: [
        {
          name: "Review",
          type: "TYPE_PROMPT",
          prompt: "Review this change.",
          command: undefined,
          workingDirectory: "repo",
          secrets: undefined,
        },
        {
          name: "Test",
          type: "TYPE_BASH",
          prompt: undefined,
          command: "make test",
          workingDirectory: "repo",
          secrets: undefined,
        },
      ],
    });
  });

  it("does not override the workspace agent during creation", () => {
    const settings = customVerificationSettings("manual", customVerificationDraft(), false);

    expect(settings.agent).toBeUndefined();
    expect(settings.startMode).toBe("START_MODE_MANUAL");
    expect(settings.steps?.map((step) => step.type)).toEqual(["TYPE_BASH"]);
  });

  it("uses concise mode copy for board rows", () => {
    expect(customVerificationModeCopy("manual").trigger).toBe("Manual start on a Verify task");
    expect(customVerificationModeCopy("automatic").action).toBe("Run automation automatically");
  });

  it("classifies managed rows by template id and start mode", () => {
    const automations = buildColumnAutomations("verify", {
      columnTitle: "Verify",
      apps: [
        {
          id: "manual-review",
          name: "Security review",
          columnKey: "verify",
          templateId: "custom-verification",
          customVerificationMode: "manual",
        },
        {
          id: "automatic-review",
          name: "Policy review",
          columnKey: "verify",
          templateId: "custom-verification",
          customVerificationMode: "automatic",
        },
      ],
    });

    expect(automations).toEqual([
      expect.objectContaining({
        kind: "custom-verification",
        name: "Security review",
        trigger: "Manual start on a Verify task",
      }),
      expect.objectContaining({
        kind: "custom-verification",
        name: "Policy review",
        action: "Run automation automatically",
      }),
    ]);
  });

  it("opens an existing managed row in the focused editor", () => {
    const automation = buildColumnAutomations("verify", {
      columnTitle: "Verify",
      apps: [
        {
          id: "verification-1",
          name: "Security review",
          columnKey: "verify",
          templateId: "custom-verification",
        },
      ],
    })[0]!;

    expect(
      columnAutomationOpenPath(automation, {
        organizationId: "org",
        factoryKey: "workspace",
        lineId: "line",
      }),
    ).toBe("/org/workspaces/workspace/lines/line?customVerification=verification-1");
  });
});
