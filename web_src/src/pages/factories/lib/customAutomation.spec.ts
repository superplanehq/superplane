import { describe, expect, it } from "bun:test";

import { buildColumnAutomations, columnAutomationOpenPath } from "./columnAutomations";
import {
  customAutomationDraft,
  customAutomationTriggersForRun,
  newCustomAutomationTrigger,
  customAutomationModeCopy,
  customAutomationSettings,
  customAutomationSettingsForTriggers,
  customAutomationSchedule,
  customAutomationTriggers,
  customAutomationMode,
  workspaceAgentFromOnboarding,
} from "./customAutomation";
import { customAutomationIdFromSearch } from "./customAutomationPaths";

describe("custom automation settings", () => {
  it("preserves a saved schedule offset and ordered steps across trigger changes", () => {
    const automation = {
      settings: {
        startMode: "START_MODE_SCHEDULED" as const,
        schedule: { cronExpression: "30 0 9 * * MON-FRI", utcOffset: 5.75 },
        steps: [
          { name: "Test", type: "TYPE_BASH" as const, command: "make test", secrets: ["deploy"] },
          { name: "Review", type: "TYPE_PROMPT" as const, prompt: "Review the repository." },
        ],
      },
    };
    const draft = customAutomationDraft(automation);
    const schedule = customAutomationSchedule(automation);
    expect(schedule).toEqual(automation.settings.schedule);
    expect(customAutomationMode(automation.settings.startMode)).toBe("scheduled");
    const saved = customAutomationSettings("scheduled", draft, "clock", schedule);
    expect(saved.schedule).toEqual(automation.settings.schedule);
    expect(saved.startMode).toBe("START_MODE_SCHEDULED");
    expect(customAutomationSettings("manual", draft).steps).toEqual(saved.steps);
    expect(customAutomationSettings("automatic", draft).steps).toEqual(saved.steps);
    expect(customAutomationSettings("leave", draft).startMode).toBe("START_MODE_ON_LEAVE");
    expect(customAutomationMode("START_MODE_ON_LEAVE")).toBe("leave");
    expect(customAutomationModeCopy("leave", "verify").trigger).toBe("When a task leaves Verify");
    expect(
      customAutomationSchedule({
        settings: { startMode: "START_MODE_SCHEDULED", schedule: { cronExpression: "0 9 * * *" } },
      }).utcOffset,
    ).toBe(0);
  });
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
      skipSetupCommands: false,
      skipRepositoryCheckout: false,
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

  it.each([undefined, {}])("saves the draft agent and run duration without settings (%j)", (automation) => {
    const draft = customAutomationDraft(automation);
    expect(draft.components[0]?.configuration.steps).toEqual([
      { name: "Hello world", type: "bash", command: 'echo "hello world"', workingDirectory: "repo" },
    ]);
    draft.components[0]!.component = "runnerClaudeCode";
    draft.components[0]!.configuration.model = "claude-opus";
    draft.components[0]!.configuration.executionTimeoutSeconds = 7200;

    const settings = customAutomationSettings("manual", draft);

    expect(settings.agent).toEqual({ component: "runnerClaudeCode", model: "claude-opus" });
    expect(settings.executionTimeoutSeconds).toBe(7200);
    expect(settings.steps?.map((step) => step.type)).toEqual(["TYPE_BASH"]);
  });

  it.each([{ settings: {} }, { settings: { steps: [] } }])(
    "keeps draft steps empty for existing settings (%j)",
    (automation) => {
      expect(customAutomationDraft(automation).components[0]?.configuration.steps).toEqual([]);
    },
  );

  it("does not send credentials for the SuperPlane agent", () => {
    const draft = customAutomationDraft({
      settings: { agent: workspaceAgentFromOnboarding({ agentHarness: "AGENT_HARNESS_SUPERPLANE" }) },
    });

    expect(customAutomationSettings("automatic", draft).agent).toEqual({
      component: "runnerSuperPlane",
      model: "",
    });

    draft.components[0]!.configuration.credentials = { source: "hosted" };
    expect(customAutomationSettings("automatic", draft).agent?.credentialSource).toBeUndefined();
  });

  it("fills an empty create draft from the workspace agent", () => {
    const draft = customAutomationDraft({
      settings: {
        agent: workspaceAgentFromOnboarding(
          { agentHarness: "AGENT_HARNESS_CLAUDE_CODE", agentIntegrationId: "claude" },
          [{ metadata: { id: "claude", name: "team", integrationName: "claude" }, status: { state: "ready" } }],
        ),
      },
    });

    expect(draft.components[0]?.component).toBe("runnerClaudeCode");
    expect(customAutomationSettings("manual", draft).agent?.component).toBe("runnerClaudeCode");
  });

  it("uses provider priority when the saved agent integration is unavailable", () => {
    const connections = [
      { metadata: { id: "openai", name: "Codex", integrationName: "openai" }, status: { state: "ready" } },
      { metadata: { id: "claude", name: "Claude", integrationName: "claude" }, status: { state: "ready" } },
    ];
    const onboarding = { agentHarness: "AGENT_HARNESS_CLAUDE_CODE" as const, agentIntegrationId: "removed" };
    expect(workspaceAgentFromOnboarding(onboarding, connections)).toEqual({
      component: "runnerClaudeCode",
      credentialSource: "integration",
      credentialIntegrationName: "Claude",
    });
    expect(workspaceAgentFromOnboarding(onboarding, [])).toEqual({ component: "runnerSuperPlane" });
  });

  it("uses concise mode copy for board rows", () => {
    expect(customAutomationModeCopy("manual").trigger).toBe("Manual start on a Verify task");
    expect(customAutomationModeCopy("automatic").action).toBe("Run automation automatically");
    expect(customAutomationModeCopy("scheduled", "backlog").action).toBe("Creates a task");
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
          displayColumnKey: "backlog",
          templateId: "custom-verification",
          customAutomationMode: "automatic",
        },
        {
          id: "scheduled-review",
          name: "Scheduled review",
          columnKey: "verify",
          templateId: "custom-verification",
          customAutomationMode: "scheduled",
          customAutomationCronExpression: "0 9 * * *",
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
        displayColumnKey: "backlog",
        trigger: "On Verify and new commits",
        action: "Run automation automatically",
      }),
      expect.objectContaining({
        name: "Scheduled review",
        trigger: "On schedule: 0 9 * * *",
      }),
    ]);
  });

  it("names every trigger on a managed row", () => {
    const [automation] = buildColumnAutomations("backlog", {
      columnTitle: "Backlog",
      apps: [
        {
          id: "eslint",
          name: "ESLint tasks",
          columnKey: "backlog",
          templateId: "custom-verification",
          customAutomationTriggers: [{ mode: "scheduled", cronExpression: "0 9 * * *" }, { mode: "automatic" }],
        },
      ],
    });

    expect(automation?.trigger).toBe("On schedule: 0 9 * * *, When a task enters Backlog");
    expect(automation?.canRunNow).toBe(true);
  });

  it("does not offer Run for an automation that only a webhook starts", () => {
    const [automation] = buildColumnAutomations("backlog", {
      columnTitle: "Backlog",
      apps: [
        {
          id: "webhook-intake",
          name: "Webhook intake",
          columnKey: "backlog",
          templateId: "custom-verification",
          customAutomationMode: "webhook",
          customAutomationRunsOnTask: false,
        },
      ],
    });

    expect(automation?.trigger).toBe("When a webhook receives a request");
    expect(automation?.canRunNow).toBeUndefined();
  });

  it.each([
    ["backlog", "leave", false, ["backlog", "phase-0", "verify", "done"]],
    ["verify", "leave", false, ["backlog", "phase-0", "verify", "done"]],
    ["backlog", "automatic", false, ["backlog", "phase-0", "done"]],
    ["verify", "automatic", false, undefined],
    ["backlog", "manual", false, undefined],
    ["backlog", "leave", true, undefined],
    ["backlog", "automatic", true, undefined],
  ] as const)("selects task stage targets for %s/%s (scheduled=%s)", (column, mode, scheduled, targets) => {
    const [automation] = buildColumnAutomations(column, {
      columnTitle: column,
      apps: [
        {
          id: "review",
          columnKey: column,
          templateId: "custom-verification",
          customAutomationMode: mode,
          customAutomationTriggers: scheduled ? [{ mode: "scheduled" }, { mode }] : undefined,
        },
      ],
    });
    expect(automation?.taskStageTargets).toEqual(targets);
  });

  it("lets a person start a Backlog automation that runs with no task", () => {
    const [automation] = buildColumnAutomations("backlog", {
      columnTitle: "Backlog",
      apps: [
        {
          id: "column-run",
          name: "Column run",
          columnKey: "backlog",
          templateId: "custom-verification",
          customAutomationMode: "manual",
          customAutomationRunsOnTask: false,
        },
      ],
    });

    expect(automation?.canRunNow).toBe(true);
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
      }),
    ).toBe("/org/workspaces/workspace?customAutomation=review-1&automationColumn=verify");
  });

  it("reads the setup query and accepts the previous name", () => {
    expect(customAutomationIdFromSearch("?customAutomation=new")).toBeNull();
    expect(customAutomationIdFromSearch("?customAutomation=review-1")).toBe("review-1");
    expect(customAutomationIdFromSearch("?customVerification=review-1")).toBe("review-1");
    expect(customAutomationIdFromSearch("")).toBeUndefined();
  });
});

describe("trigger run context", () => {
  it.each([
    [undefined, ["manual", "automatic", "leave", "scheduled", "webhook"]],
    [true, ["manual", "automatic", "leave"]],
    [false, ["manual", "scheduled", "webhook"]],
  ] as const)("filters triggers for runsOnTask=%s", (runsOnTask, expected) => {
    const triggers = (["manual", "automatic", "leave", "scheduled", "webhook"] as const).map((mode) =>
      newCustomAutomationTrigger(mode),
    );
    expect(customAutomationTriggersForRun(triggers, runsOnTask).map(({ mode }) => mode)).toEqual([...expected]);
  });
});

describe("webhook triggers", () => {
  it("saves a single webhook trigger with its webhook and reads it back", () => {
    const trigger = newCustomAutomationTrigger("webhook", customAutomationSchedule(), "webhook-1");
    const settings = customAutomationSettingsForTriggers([trigger], customAutomationDraft({}), "webhook", false);

    expect(settings.startMode).toBe("START_MODE_WEBHOOK");
    expect(settings.runsOnTask).toBe(false);
    expect(settings.triggers).toEqual([
      expect.objectContaining({ startMode: "START_MODE_WEBHOOK", webhookId: "webhook-1" }),
    ]);
    expect(customAutomationTriggers({ settings })).toEqual([
      expect.objectContaining({ mode: "webhook", webhookId: "webhook-1" }),
    ]);
  });
});
