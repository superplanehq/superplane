import type {
  FactoriesFactoryCustomAutomation,
  FactoriesFactoryCustomAutomationSettings,
  FactoriesFactoryCustomAutomationStep,
  FactoriesFactoryOnboarding,
  OrganizationsIntegration,
  FactoryCustomAutomationStartTrigger,
  FactoryCustomAutomationStartMode,
} from "@/api-client";

import type { PlanningReviewDraft, PlanningReviewStep } from "../pages/planningReviewMockup";

export const CUSTOM_AUTOMATION_TEMPLATE_ID = "custom-verification";

export type CustomAutomationMode = "manual" | "automatic" | "scheduled" | "leave";
const CUSTOM_AUTOMATION_START_MODES: Record<CustomAutomationMode, FactoryCustomAutomationStartMode> = {
  manual: "START_MODE_MANUAL",
  automatic: "START_MODE_AUTOMATIC",
  scheduled: "START_MODE_SCHEDULED",
  leave: "START_MODE_ON_LEAVE",
};
export type CustomAutomationSchedule = { cronExpression: string; utcOffset: number };
export type CustomAutomationColumn = "backlog" | "implement" | "verify" | "done" | `custom-${string}`;

export function customAutomationColumn(value: string | null | undefined): CustomAutomationColumn {
  if (
    value === "backlog" ||
    value === "implement" ||
    value === "done" ||
    value === "verify" ||
    value?.startsWith("custom-")
  ) {
    return value as CustomAutomationColumn;
  }
  return "verify";
}

export function customAutomationColumnLabel(column: CustomAutomationColumn): string {
  if (column.startsWith("custom-")) return "this stage";
  switch (column) {
    case "backlog":
      return "Backlog";
    case "implement":
      return "Implement";
    case "done":
      return "Done";
    default:
      return "Verify";
  }
}
export type CustomAutomationTrigger = {
  id: string;
  publicId?: string;
  mode: CustomAutomationMode;
  schedule: CustomAutomationSchedule;
};

export function newCustomAutomationTrigger(
  mode: CustomAutomationMode = "manual",
  schedule: CustomAutomationSchedule = customAutomationSchedule(),
): CustomAutomationTrigger {
  return { id: crypto.randomUUID(), mode, schedule };
}

export function customAutomationTriggers(automation?: FactoriesFactoryCustomAutomation): CustomAutomationTrigger[] {
  const saved = automation?.settings?.triggers;
  if (saved && saved.length > 0) {
    return saved.map((trigger) =>
      newCustomAutomationTrigger(
        customAutomationMode(trigger.startMode),
        trigger.startMode === "START_MODE_SCHEDULED"
          ? {
              cronExpression: trigger.schedule?.cronExpression ?? "",
              utcOffset: trigger.schedule?.utcOffset ?? 0,
            }
          : customAutomationSchedule(),
      ),
    );
  }
  return [
    newCustomAutomationTrigger(
      customAutomationMode(automation?.settings?.startMode),
      customAutomationSchedule(automation),
    ),
  ];
}

export function customAutomationAllowsManualStart(settings?: FactoriesFactoryCustomAutomationSettings): boolean {
  if (settings?.runsOnTask === false) {
    return false;
  }
  if (settings?.triggers && settings.triggers.length > 0) {
    return settings.triggers.some((trigger) => trigger.startMode === "START_MODE_MANUAL");
  }
  return customAutomationMode(settings?.startMode) === "manual";
}

export function customAutomationRunsOnTask(
  saved: boolean | undefined,
  triggers: Array<{ mode: CustomAutomationMode }>,
): boolean {
  if (saved !== undefined) {
    return saved;
  }
  if (triggers.some((trigger) => trigger.mode === "automatic" || trigger.mode === "leave")) {
    return true;
  }
  if (triggers.some((trigger) => trigger.mode === "scheduled")) {
    return false;
  }
  return true;
}

export function customAutomationTriggersForRun(
  triggers: CustomAutomationTrigger[],
  runsOnTask: boolean | undefined,
): CustomAutomationTrigger[] {
  if (runsOnTask === undefined) return triggers;
  return triggers.filter((trigger) => {
    if (runsOnTask) return trigger.mode !== "scheduled";
    return trigger.mode !== "automatic" && trigger.mode !== "leave";
  });
}

export function customAutomationSchedule(automation?: FactoriesFactoryCustomAutomation): CustomAutomationSchedule {
  if (automation?.settings?.startMode === "START_MODE_SCHEDULED") {
    return {
      cronExpression: automation.settings.schedule?.cronExpression ?? "",
      utcOffset: automation.settings.schedule?.utcOffset ?? 0,
    };
  }
  return { cronExpression: "0 9 * * *", utcOffset: -new Date().getTimezoneOffset() / 60 };
}
export type ManualCustomAutomation = {
  id: string;
  name: string;
  icon?: string;
  columnKey?: CustomAutomationColumn;
};

export const AUTOMATION_ICONS = [
  "sparkles",
  "shield-check",
  "search",
  "bug",
  "file-code",
  "code",
  "terminal",
  "eye",
  "gauge",
  "git-pull-request",
  "cpu",
  "circle-check",
  "activity",
  "alarm-clock",
  "archive",
  "bell",
  "bolt",
  "book",
  "box",
  "boxes",
  "braces",
  "calendar",
  "camera",
  "chart-line",
  "check",
  "clock",
  "cloud",
  "cog",
  "component",
  "container",
  "database",
  "factory",
  "file-json",
  "file-text",
  "fingerprint",
  "flag",
  "folder",
  "git-branch",
  "git-commit",
  "git-merge",
  "globe",
  "key",
  "layers",
  "lock",
  "message-square",
  "network",
  "package",
  "rocket",
] as const;

export const DEFAULT_CUSTOM_AUTOMATION_ICON = "sparkles";
export const DEFAULT_CUSTOM_AUTOMATION_TIMEOUT_SECONDS = 3600;

export function customAutomationMode(value: FactoryCustomAutomationStartMode | undefined): CustomAutomationMode {
  if (value === "START_MODE_SCHEDULED") return "scheduled";
  if (value === "START_MODE_ON_LEAVE") return "leave";
  return value === "START_MODE_AUTOMATIC" ? "automatic" : "manual";
}

export function customAutomationModeCopy(
  mode: CustomAutomationMode,
  columnKey: CustomAutomationColumn = "verify",
): {
  trigger: string;
  action: string;
} {
  if (mode === "scheduled") {
    return {
      trigger: "On schedule",
      action: columnKey === "backlog" ? "Creates a task" : "Run automation on the repository default branch",
    };
  }
  if (mode === "leave") {
    return {
      trigger: `When a task leaves ${customAutomationColumnLabel(columnKey)}`,
      action: "Run automation automatically",
    };
  }
  if (mode === "automatic") {
    return {
      trigger:
        columnKey === "verify"
          ? "On Verify and new commits"
          : `When a task enters ${customAutomationColumnLabel(columnKey)}`,
      action: "Run automation automatically",
    };
  }
  return {
    trigger: `Manual start on a ${customAutomationColumnLabel(columnKey)} task`,
    action: "Run automation on demand",
  };
}

const integrationAgentComponents = new Map([
  ["claude", "runnerClaudeCode"],
  ["openai", "runnerCodex"],
  ["openrouter", "runnerOpenRouter"],
  ["customLlm", "runnerOpenRouter"],
]);

export function workspaceAgentFromOnboarding(
  onboarding?: FactoriesFactoryOnboarding,
  connections?: OrganizationsIntegration[],
): FactoriesFactoryCustomAutomationSettings["agent"] {
  if (!onboarding) return undefined;
  if (onboarding.codexSubscription) return { component: "runnerCodex", credentialSource: "subscription" };
  const hosted = { component: "runnerSuperPlane" };
  if (!onboarding.agentIntegrationId && onboarding.agentHarness === "AGENT_HARNESS_SUPERPLANE") return hosted;
  if (!connections) return undefined;
  const ready = connections.filter(
    ({ metadata, status }) =>
      status?.state === "ready" && integrationAgentComponents.has(metadata?.integrationName ?? ""),
  );
  let selected = ready.find(({ metadata }) => metadata?.id === onboarding.agentIntegrationId);
  if (!selected && onboarding.agentHarness !== "AGENT_HARNESS_SUPERPLANE") {
    // Use the same integration priority as existing custom automations in older workspaces.
    for (const provider of integrationAgentComponents.keys()) {
      selected = ready.find(({ metadata }) => metadata?.integrationName === provider);
      if (selected) break;
    }
  }
  const component = integrationAgentComponents.get(selected?.metadata?.integrationName ?? "");
  if (!component) return hosted;
  return { component, credentialSource: "integration", credentialIntegrationName: selected?.metadata?.name };
}

export function customAutomationDraft(
  automation?: FactoriesFactoryCustomAutomation,
  customProvider = false,
): PlanningReviewDraft {
  const settings = automation?.settings;
  const agent = settings?.agent;
  const steps = settings
    ? (settings.steps ?? []).map(stepFromApi)
    : [
        {
          name: "Hello world",
          type: "bash" as const,
          command: 'echo "hello world"',
          workingDirectory: "repo",
        },
      ];
  return {
    title: automation?.automation?.name?.trim() || "Custom automation",
    components: [
      {
        id: "custom-automation-agent",
        title: "Automation agent",
        description: "Run the configured automation steps.",
        expanded: true,
        component: agent?.component,
        configuration: {
          model: agent?.model ?? "",
          credentials: customAutomationCredentials(agent),
          ...(customProvider ? { llmProvider: "custom" } : {}),
          steps,
          executionTimeoutSeconds: customAutomationTimeout(settings?.executionTimeoutSeconds),
          skipSetupCommands: settings?.skipSetupCommands === true,
          skipRepositoryCheckout: settings?.skipRepositoryCheckout === true,
        },
        concurrency: { max: "5", key: "" },
      },
    ],
  };
}

export function customAutomationCredentials(agent: FactoriesFactoryCustomAutomationSettings["agent"]) {
  return agent?.credentialSource
    ? { source: agent.credentialSource, integration: { name: agent.credentialIntegrationName } }
    : undefined;
}

export function customAutomationSettings(
  mode: CustomAutomationMode,
  draft: PlanningReviewDraft,
  icon = DEFAULT_CUSTOM_AUTOMATION_ICON,
  schedule?: CustomAutomationSchedule,
): FactoriesFactoryCustomAutomationSettings {
  const component = draft.components[0];
  const configuration = component?.configuration ?? {};
  const steps = Array.isArray(configuration.steps) ? (configuration.steps as PlanningReviewStep[]) : [];
  const credentials = configuration.credentials as { source?: string; integration?: { name?: string } } | undefined;
  const superPlaneAgent = component?.component === "runnerSuperPlane";
  return {
    startMode: CUSTOM_AUTOMATION_START_MODES[mode],
    ...(mode === "scheduled" ? { schedule } : {}),
    icon,
    executionTimeoutSeconds: customAutomationTimeout(configuration.executionTimeoutSeconds),
    skipSetupCommands: configuration.skipSetupCommands === true,
    skipRepositoryCheckout: configuration.skipRepositoryCheckout === true,
    agent: component?.component
      ? {
          component: component.component,
          model: typeof configuration.model === "string" ? configuration.model : "",
          ...(!superPlaneAgent && credentials?.source
            ? { credentialSource: credentials.source, credentialIntegrationName: credentials.integration?.name }
            : {}),
        }
      : undefined,
    steps: steps.map(stepToApi),
  };
}

function customAutomationTimeout(value: unknown): number {
  return typeof value === "number" && value > 0 ? value : DEFAULT_CUSTOM_AUTOMATION_TIMEOUT_SECONDS;
}

function stepFromApi(step: FactoriesFactoryCustomAutomationStep): PlanningReviewStep {
  return {
    name: step.name ?? "",
    type: step.type === "TYPE_BASH" ? "bash" : "prompt",
    prompt: step.prompt,
    command: step.command,
    workingDirectory: step.workingDirectory,
    secrets: step.secrets,
  };
}

export function customAutomationSettingsForTriggers(
  triggers: CustomAutomationTrigger[],
  draft: PlanningReviewDraft,
  icon = DEFAULT_CUSTOM_AUTOMATION_ICON,
  runsOnTask = true,
): FactoriesFactoryCustomAutomationSettings {
  const active = customAutomationTriggersForRun(triggers, runsOnTask);
  const current = active.length > 0 ? active : [newCustomAutomationTrigger()];
  const scheduled = current.filter((trigger) => trigger.mode === "scheduled");
  const primary = current.some((trigger) => trigger.mode === "automatic")
    ? "automatic"
    : current.some((trigger) => trigger.mode === "leave")
      ? "leave"
      : scheduled.length > 0
        ? "scheduled"
        : "manual";
  const settings = customAutomationSettings(primary, draft, icon, scheduled[0]?.schedule);
  const chosen =
    current.length === 1
      ? customAutomationSettings(current[0]!.mode, draft, icon, current[0]!.schedule)
      : {
          ...settings,
          schedule: scheduled.length === 1 ? scheduled[0]!.schedule : undefined,
          triggers: current.map(triggerToApi),
        };
  return { ...chosen, runsOnTask };
}

function triggerToApi(trigger: CustomAutomationTrigger): FactoryCustomAutomationStartTrigger {
  return {
    startMode: CUSTOM_AUTOMATION_START_MODES[trigger.mode],
    schedule: trigger.mode === "scheduled" ? trigger.schedule : undefined,
  };
}

function stepToApi(step: PlanningReviewStep): FactoriesFactoryCustomAutomationStep {
  return {
    name: step.name,
    type: step.type === "bash" ? "TYPE_BASH" : "TYPE_PROMPT",
    prompt: step.type === "prompt" ? (step.prompt ?? "") : undefined,
    command: step.type === "bash" ? (step.command ?? "") : undefined,
    workingDirectory: step.workingDirectory,
    secrets: step.secrets,
  };
}
