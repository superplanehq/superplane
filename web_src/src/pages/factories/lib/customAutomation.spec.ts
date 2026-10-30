import { describe, expect, it } from "bun:test";

import { buildColumnAutomations, columnAutomationOpenPath } from "./columnAutomations";
import {
  customAutomationDraft,
  customAutomationModeCopy,
  customAutomationSettings,
  workspaceAgentFromOnboarding,
} from "./customAutomation";
import { customAutomationIdFromSearch } from "./customAutomationPaths";

describe("custom automation settings", () => {
  it("maps saved agent and ordered steps into the shared editor and back", () => {
    const draft = customAutomationDraft({
      automation: { id: "review-1", name: "Security review" },
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
    expect(customAutomationSettings("automatic", draft)).toEqual({
      startMode: "START_MODE_AUTOMATIC",
      icon: "sparkles",
      executionTimeoutSeconds: 3600,
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

  it("saves the draft agent and run duration", () => {
    const draft = customAutomationDraft();
    draft.components[0]!.component = "runnerClaudeCode";
    draft.components[0]!.configuration.model = "claude-opus";
    draft.components[0]!.configuration.executionTimeoutSeconds = 7200;

    const settings = customAutomationSettings("manual", draft);

    expect(settings.agent).toEqual({ component: "runnerClaudeCode", model: "claude-opus" });
    expect(settings.executionTimeoutSeconds).toBe(7200);
    expect(settings.steps?.map((step) => step.type)).toEqual(["TYPE_BASH"]);
  });

  it("does not send credentials for the SuperPlane agent", () => {
    const draft = customAutomationDraft({
      settings: { agent: workspaceAgentFromOnboarding({ agentHarness: "AGENT_HARNESS_SUPERPLANE" }) },
    });

    expect(customAutomationSettings("automatic", draft).agent).toEqual({
      component: "runnerSuperPlane",
      model: "",
    });

    draft.components[0]!.configuration.credentialSource = "hosted";
    expect(customAutomationSettings("automatic", draft).agent?.credentialSource).toBeUndefined();
  });

  it("fills an empty create draft from the workspace agent", () => {
    const draft = customAutomationDraft({
      settings: { agent: workspaceAgentFromOnboarding({ agentHarness: "AGENT_HARNESS_CLAUDE_CODE" }) },
    });

    expect(draft.components[0]?.component).toBe("runnerClaudeCode");
    expect(customAutomationSettings("manual", draft).agent?.component).toBe("runnerClaudeCode");
  });

  it("uses concise mode copy for board rows", () => {
    expect(customAutomationModeCopy("manual").trigger).toBe("Manual start on a Verify task");
    expect(customAutomationModeCopy("automatic").action).toBe("Run automation automatically");
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
          customAutomationMode: "manual",
        },
        {
          id: "automatic-review",
          name: "Policy review",
          columnKey: "verify",
          templateId: "custom-verification",
          customAutomationMode: "automatic",
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
          id: "review-1",
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
    ).toBe("/org/workspaces/workspace/lines/line?customAutomation=review-1");
  });

  it("reads the setup query and accepts the previous name", () => {
    expect(customAutomationIdFromSearch("?customAutomation=new")).toBeNull();
    expect(customAutomationIdFromSearch("?customAutomation=review-1")).toBe("review-1");
    expect(customAutomationIdFromSearch("?customVerification=review-1")).toBe("review-1");
    expect(customAutomationIdFromSearch("")).toBeUndefined();
  });
});
