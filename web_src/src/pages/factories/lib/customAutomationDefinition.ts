import { CORE_SCHEMA, dump, load, YAMLException } from "js-yaml";
import type { FactoriesFactoryAppTemplateAgent } from "@/api-client";
import { hostedModelForPicker, type AutomationHostedModels } from "./customAutomationHostedModels";
import type { PlanningReviewStep } from "../pages/planningReviewMockup";
import {
  AUTOMATION_ICONS,
  customAutomationDraft,
  customAutomationSchedule,
  DEFAULT_CUSTOM_AUTOMATION_ICON,
  type CustomAutomationColumn,
  type CustomAutomationTrigger,
} from "./customAutomation";

type AutomationHarness = "superplane" | "claude-code" | "codex" | "openrouter";
type AutomationCredentials = { source: "integration"; name: string } | { source: "subscription" | "hosted" };
type AutomationAgent = {
  harness?: AutomationHarness;
  model?: string;
  credentials?: AutomationCredentials;
  additional_instructions?: string;
  thinking_level?: string;
  timeout?: string;
  skip_setup_commands?: boolean;
  skip_repository_checkout?: boolean;
};
type AutomationTrigger =
  | { id?: string; uses: "manual" | "task_enters_stage" | "task_leaves_stage" | "pull_request_updated" }
  | { id?: string; uses: "schedule"; config: { cron: string; timezone?: string } }
  | { id?: string; uses: "webhook"; config: { webhook_id: string } };
type AutomationStep = {
  name: string;
  bash?: string;
  prompt?: string;
  secrets?: string[];
  working_directory?: string;
};
export type AutomationDefinition = {
  apiVersion: "v2";
  kind: "Automation";
  name: string;
  icon?: string;
  timeout?: string;
  run_context?: "task" | "workspace";
  agent?: AutomationAgent;
  triggers: AutomationTrigger[];
  steps: AutomationStep[];
};

export function newAutomationDefinition(stage: CustomAutomationColumn): AutomationDefinition {
  return {
    apiVersion: "v2",
    kind: "Automation",
    name: "Custom automation",
    triggers: [{ uses: stage === "verify" ? "manual" : "task_enters_stage" }],
    steps: [],
  };
}

export function automationDefinitionYaml(definition: AutomationDefinition): string {
  return dump(definition, { schema: CORE_SCHEMA, lineWidth: -1, noRefs: true, quotingType: '"' });
}

export function parseAutomationForCanvas(source: string, stage: CustomAutomationColumn): AutomationDefinition {
  let value: unknown;
  try {
    value = load(source, {
      schema: CORE_SCHEMA,
      listener(event, state) {
        // Aliases return a value without a scalar, sequence, or mapping of their own.
        if (event === "close" && !state.kind && state.result != null) {
          throw new Error("Replace YAML aliases with their values.");
        }
      },
    });
  } catch (error) {
    if (!(error instanceof YAMLException)) throw error;
    throw new Error(`Line ${error.mark.line + 1}, column ${error.mark.column + 1} · ${error.reason}`, {
      cause: error,
    });
  }
  const document = mapping(value, "Automation", [
    "apiVersion",
    "kind",
    "name",
    "icon",
    "timeout",
    "run_context",
    "agent",
    "triggers",
    "steps",
  ]);
  if (document.apiVersion !== "v2") problem("apiVersion", "Use v2.");
  if (document.kind !== "Automation") problem("kind", "Use Automation.");
  const name = singleLine(document.name, "name");
  const icon = optionalText(document.icon, "icon");
  if (icon !== undefined && !AUTOMATION_ICONS.some((key) => key === icon)) {
    problem("icon", "Choose an icon from the automation icon picker.");
  }
  const timeout = optionalText(document.timeout, "timeout");
  automationTimeoutSeconds(timeout, "timeout");
  const triggers = nonemptyList(document.triggers, "triggers").map((trigger, index) =>
    parseTrigger(trigger, `triggers[${index}]`, stage),
  );
  validateTriggerIdentities(triggers);
  const runContext = document.run_context;
  if (runContext !== undefined && runContext !== "task" && runContext !== "workspace") {
    problem("run_context", "Use task or workspace.");
  }
  const taskEvents = triggers.some(
    ({ uses }) => uses === "task_enters_stage" || uses === "task_leaves_stage" || uses === "pull_request_updated",
  );
  const workspaceEvents = triggers.some(({ uses }) => uses === "schedule" || uses === "webhook");
  if (taskEvents && triggers.some(({ uses }) => uses === "schedule"))
    problem("triggers", "Task events cannot share an automation with schedule triggers.");
  if (runContext === "task" && workspaceEvents)
    problem("run_context", "Schedule and webhook triggers require workspace context.");
  if (runContext === "workspace" && taskEvents) problem("run_context", "Task event triggers require task context.");
  const agent = document.agent === undefined ? undefined : parseAgent(document.agent);
  if (
    timeout !== undefined &&
    agent?.timeout !== undefined &&
    automationTimeoutSeconds(timeout, "timeout") !== automationTimeoutSeconds(agent.timeout, "agent.timeout")
  ) {
    problem("agent.timeout", "Use one timeout value for this automation.");
  }
  const steps = document.steps;
  if (!Array.isArray(steps)) problem("steps", "Enter a list of steps.");
  return {
    apiVersion: "v2",
    kind: "Automation",
    name,
    icon,
    timeout,
    run_context: runContext,
    agent,
    triggers,
    steps: steps.map(parseStep),
  };
}

function automationTimeoutSeconds(value: string | undefined, path: string): number | undefined {
  if (value === undefined) return undefined;
  const units: Record<string, number> = {
    ns: 1,
    us: 1000,
    µs: 1000,
    μs: 1000,
    ms: 1000000,
    s: 1000000000,
    m: 60000000000,
    h: 3600000000000,
  };
  let remaining = value.replace(/^[+]/, "");
  if (remaining === "0") return 0;
  if (!remaining) problem(path, "Enter a duration such as 2h or 30s.");
  let nanoseconds = 0;
  while (remaining) {
    const part = remaining.match(/^((?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))(ns|us|µs|μs|ms|s|m|h)/);
    if (!part) problem(path, "Use a duration from 0s through 24h in whole seconds, such as 2h or 30s.");
    nanoseconds += Math.trunc(Number(part[1]) * units[part[2]]!);
    remaining = remaining.slice(part[0].length);
  }
  const seconds = nanoseconds / 1000000000;
  if (!Number.isSafeInteger(nanoseconds) || !Number.isInteger(seconds) || seconds < 0 || seconds > 86400) {
    problem(path, "Use a duration from 0s through 24h in whole seconds, such as 2h or 30s.");
  }
  return seconds;
}

export function automationTimeout(seconds: number): string {
  if (seconds !== 0 && seconds % 3600 === 0) return `${seconds / 3600}h`;
  if (seconds !== 0 && seconds % 60 === 0) return `${seconds / 60}m`;
  return `${seconds}s`;
}

function parseTrigger(value: unknown, path: string, stage: CustomAutomationColumn): AutomationTrigger {
  const trigger = mapping(value, path, ["id", "uses", "config"]);
  const id = optionalText(trigger.id, `${path}.id`);
  switch (trigger.uses) {
    case "schedule":
      return parseScheduleTrigger(trigger.config, id, path);
    case "webhook":
      return parseWebhookTrigger(trigger.config, id, path);
    case "pull_request_updated":
      if (stage !== "verify") problem(`${path}.uses`, "This trigger requires Verify.");
      break;
    case "task_enters_stage":
      if (stage === "verify") problem(`${path}.uses`, "Use pull_request_updated in Verify.");
      break;
    case "manual":
    case "task_leaves_stage":
      break;
    default:
      problem(`${path}.uses`, "Choose a supported trigger.");
  }
  if (trigger.config !== undefined) problem(`${path}.config`, "Only schedule and webhook accept a configuration.");
  return { id, uses: trigger.uses };
}

function parseScheduleTrigger(value: unknown, id: string | undefined, path: string): AutomationTrigger {
  const config = mapping(value, `${path}.config`, ["cron", "timezone"]);
  const cron = singleLine(config.cron, `${path}.config.cron`);
  if (![5, 6].includes(cron.trim().split(/\s+/).length)) {
    problem(`${path}.config.cron`, "Enter a cron expression with five or six fields.");
  }
  const timezone = optionalText(config.timezone, `${path}.config.timezone`);
  if (
    timezone !== undefined &&
    (!/^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$/.test(timezone) ||
      !Number.isFinite(Number(timezone)) ||
      Number(timezone) < -12 ||
      Number(timezone) > 14)
  ) {
    problem(`${path}.config.timezone`, 'Use a UTC offset from -12 through 14, such as "5.5".');
  }
  return { id, uses: "schedule", config: { cron, timezone } };
}

function parseWebhookTrigger(value: unknown, id: string | undefined, path: string): AutomationTrigger {
  const config = mapping(value, `${path}.config`, ["webhook_id"]);
  return { id, uses: "webhook", config: { webhook_id: singleLine(config.webhook_id, `${path}.config.webhook_id`) } };
}

function validateTriggerIdentities(triggers: AutomationTrigger[]) {
  const ids = new Set<string>();
  const events = new Set<string>();
  const webhooks = new Set<string>();
  triggers.forEach((trigger, index) => {
    const { id, uses } = trigger;
    const path = `triggers[${index}]`;
    if (trigger.uses === "webhook") {
      if (webhooks.has(trigger.config.webhook_id)) problem(`${path}.config.webhook_id`, "Use each webhook once.");
      webhooks.add(trigger.config.webhook_id);
    } else if (uses !== "schedule") {
      if (events.has(uses)) problem(`${path}.uses`, "Use this trigger once.");
      events.add(uses);
    }
    if (id) {
      if (ids.has(id)) problem(`${path}.id`, "Use a unique trigger ID.");
      ids.add(id);
    }
  });
}

function parseAgent(value: unknown): AutomationAgent {
  const entry = mapping(value, "agent", [
    "harness",
    "model",
    "credentials",
    "additional_instructions",
    "thinking_level",
    "timeout",
    "skip_setup_commands",
    "skip_repository_checkout",
  ]);
  const harness = entry.harness;
  if (
    harness !== undefined &&
    harness !== "superplane" &&
    harness !== "claude-code" &&
    harness !== "codex" &&
    harness !== "openrouter"
  ) {
    problem("agent.harness", "Choose a supported agent.");
  }
  if (entry.credentials !== undefined && harness === undefined)
    problem("agent.harness", "Select a harness for explicit credentials.");
  if (entry.credentials !== undefined && harness === "superplane")
    problem("agent.credentials", "SuperPlane uses workspace credentials.");
  if (entry.additional_instructions !== undefined && typeof entry.additional_instructions !== "string")
    problem("agent.additional_instructions", "Enter a string.");
  const timeout = optionalText(entry.timeout, "agent.timeout");
  automationTimeoutSeconds(timeout, "agent.timeout");
  return {
    harness,
    model: optionalText(entry.model, "agent.model"),
    credentials: entry.credentials === undefined ? undefined : parseAgentCredentials(entry.credentials, harness!),
    additional_instructions: entry.additional_instructions,
    thinking_level: optionalText(entry.thinking_level, "agent.thinking_level"),
    timeout,
    skip_setup_commands: optionalBoolean(entry.skip_setup_commands, "agent.skip_setup_commands"),
    skip_repository_checkout: optionalBoolean(entry.skip_repository_checkout, "agent.skip_repository_checkout"),
  };
}

function parseAgentCredentials(value: unknown, harness: AutomationHarness): AutomationCredentials {
  const credentials = mapping(value, "agent.credentials", ["source", "name"]);
  if (credentials.source === "integration") {
    return { source: "integration", name: singleLine(credentials.name, "agent.credentials.name") };
  }
  if (credentials.source === "subscription") {
    if (harness !== "codex") problem("agent.credentials.source", "A ChatGPT subscription requires Codex.");
    if (credentials.name !== undefined) problem("agent.credentials.name", "Remove the subscription name.");
    return { source: "subscription" };
  }
  if (credentials.source === "hosted") {
    if (credentials.name !== undefined) problem("agent.credentials.name", "Remove the hosted connection name.");
    return { source: "hosted" };
  }
  return problem("agent.credentials.source", "Use integration, hosted, or subscription.");
}

function parseStep(value: unknown, index: number): AutomationStep {
  const path = `steps[${index}]`;
  const entry = mapping(value, path, ["name", "bash", "prompt", "secrets", "working_directory"]);
  const name = singleLine(entry.name, `${path}.name`);
  const bash = optionalText(entry.bash, `${path}.bash`);
  const prompt = optionalText(entry.prompt, `${path}.prompt`);
  if ((bash === undefined) === (prompt === undefined)) {
    problem(path, "Enter either bash or prompt.");
  }
  const working_directory = optionalText(entry.working_directory, `${path}.working_directory`);
  if (entry.secrets === undefined) return { name, bash, prompt, working_directory };
  if (!bash) problem(`${path}.secrets`, "Only Bash steps can reference secrets.");
  if (!Array.isArray(entry.secrets)) problem(`${path}.secrets`, "Enter a list of secret names.");
  const secrets = entry.secrets.map((secret, i) => singleLine(secret, `${path}.secrets[${i}]`));
  if (secrets.length !== new Set(secrets).size) problem(`${path}.secrets`, "Use each secret name once.");
  return { name, bash, secrets, working_directory };
}

function mapping(value: unknown, path: string, fields: string[]): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) problem(path, "Enter a mapping.");
  for (const field of Object.keys(value)) {
    if (!fields.includes(field)) {
      const replacements: Record<string, string> = {
        use: "uses",
        utc_offset: "timezone",
        timeout_seconds: "timeout",
        execution_timeout_seconds: "timeout",
        runs_on_task: "run_context",
        instructions: "additional_instructions",
      };
      const replacement =
        path === "Automation" && (field === "skip_setup_commands" || field === "skip_repository_checkout")
          ? `agent.${field}`
          : replacements[field];
      problem(
        `${path}.${field}`,
        replacement ? `Use ${replacement} instead of ${field}.` : "Remove the unknown field.",
      );
    }
  }
  return value as Record<string, unknown>;
}

function nonemptyList(value: unknown, path: string): unknown[] {
  if (!Array.isArray(value) || !value.length) problem(path, "Enter a list with at least one item.");
  return value;
}

function text(value: unknown, path: string): string {
  if (typeof value !== "string" || !value.trim()) problem(path, "Enter a nonblank string.");
  return value;
}

function optionalText(value: unknown, path: string): string | undefined {
  return value === undefined ? undefined : text(value, path);
}

function optionalBoolean(value: unknown, path: string): boolean | undefined {
  if (value === undefined) return undefined;
  if (typeof value !== "boolean") problem(path, "Use true or false.");
  return value;
}

function singleLine(value: unknown, path: string): string {
  const string = text(value, path);
  if (/[\r\n]/.test(string)) problem(path, "Enter a single line to use the visual editor.");
  return string;
}

function problem(path: string, reason: string): never {
  throw new Error(`${path} · ${reason}`);
}

export function automationDefinitionCanvas(
  definition: AutomationDefinition,
  agent?: FactoriesFactoryAppTemplateAgent,
  customProvider = false,
  hosted?: AutomationHostedModels,
) {
  const draft = customAutomationDraft(
    {
      settings: {
        agent: { ...agent, model: hostedModelForPicker(definition.agent?.model, hosted) },
        executionTimeoutSeconds: automationTimeoutSeconds(definition.timeout ?? definition.agent?.timeout, "timeout"),
        skipSetupCommands: definition.agent?.skip_setup_commands,
        skipRepositoryCheckout: definition.agent?.skip_repository_checkout,
        steps: definition.steps.map((step) => ({
          name: step.name,
          type: step.bash !== undefined ? "TYPE_BASH" : "TYPE_PROMPT",
          command: step.bash,
          prompt: step.prompt,
          secrets: step.secrets,
          workingDirectory: step.working_directory,
        })),
      },
    },
    customProvider,
  );
  return {
    name: definition.name,
    icon: definition.icon ?? DEFAULT_CUSTOM_AUTOMATION_ICON,
    runsOnTask: automationDefinitionRunContext(definition),
    triggers: definition.triggers.map(canvasTrigger),
    draft,
  };
}

function automationDefinitionRunContext(definition: AutomationDefinition): boolean | undefined {
  if (definition.run_context !== undefined) return definition.run_context === "task";
  const withoutTask = ({ uses }: AutomationTrigger) => uses === "schedule" || uses === "webhook";
  if (!definition.triggers.some(withoutTask)) return true;
  if (definition.triggers.some(({ uses }) => uses === "schedule") || definition.triggers.every(withoutTask))
    return false;
  // A webhook can share the automation with task-bound triggers.
  return undefined;
}

function canvasTrigger(trigger: AutomationTrigger, index: number): CustomAutomationTrigger {
  return {
    id: trigger.id === undefined ? `implicit:${index}` : `public:${trigger.id}`,
    publicId: trigger.id,
    mode:
      trigger.uses === "schedule"
        ? "scheduled"
        : trigger.uses === "webhook"
          ? "webhook"
          : trigger.uses === "manual"
            ? "manual"
            : trigger.uses === "task_leaves_stage"
              ? "leave"
              : "automatic",
    schedule:
      trigger.uses === "schedule"
        ? { cronExpression: trigger.config.cron, utcOffset: Number(trigger.config.timezone ?? "0") }
        : customAutomationSchedule(),
    ...(trigger.uses === "webhook" ? { webhookId: trigger.config.webhook_id } : {}),
  };
}

export function automationDefinitionTriggers(
  triggers: CustomAutomationTrigger[],
  stage: CustomAutomationColumn,
): AutomationTrigger[] {
  return triggers.map(({ publicId, mode, schedule, webhookId }) => {
    if (mode === "webhook") return { id: publicId, uses: "webhook", config: { webhook_id: webhookId ?? "" } };
    if (mode === "scheduled") {
      return {
        id: publicId,
        uses: "schedule",
        config: { cron: schedule.cronExpression, timezone: String(schedule.utcOffset) },
      };
    }
    if (mode === "manual") return { id: publicId, uses: "manual" };
    if (mode === "leave") return { id: publicId, uses: "task_leaves_stage" };
    return { id: publicId, uses: stage === "verify" ? "pull_request_updated" : "task_enters_stage" };
  });
}

export function automationDefinitionSteps(steps: PlanningReviewStep[]): AutomationStep[] {
  return steps.map((step) => ({
    name: step.name,
    bash: step.type === "bash" ? (step.command ?? "") : undefined,
    prompt: step.type === "prompt" ? (step.prompt ?? "") : undefined,
    secrets: step.type === "bash" ? step.secrets : undefined,
    working_directory: step.workingDirectory,
  }));
}
