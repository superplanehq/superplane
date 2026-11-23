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
};
type AutomationTrigger =
  | { id?: string; use: "manual" | "task_enters_stage" | "task_leaves_stage" | "pull_request_updated" }
  | { id?: string; use: "schedule"; config: { cron: string; utc_offset?: number } };
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
  timeout_seconds?: number;
  runs_on_task?: boolean;
  skip_setup_commands?: boolean;
  skip_repository_checkout?: boolean;
  agent?: AutomationAgent;
  triggers: AutomationTrigger[];
  steps: AutomationStep[];
};

export function newAutomationDefinition(stage: CustomAutomationColumn): AutomationDefinition {
  return {
    apiVersion: "v2",
    kind: "Automation",
    name: "Custom automation",
    triggers: [{ use: stage === "verify" ? "manual" : "task_enters_stage" }],
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
    "timeout_seconds",
    "runs_on_task",
    "skip_setup_commands",
    "skip_repository_checkout",
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
  const timeout = document.timeout_seconds;
  if (
    timeout !== undefined &&
    (typeof timeout !== "number" || !Number.isInteger(timeout) || timeout < 1 || timeout > 86400)
  ) {
    problem("timeout_seconds", "Enter an integer from 1 through 86400.");
  }
  const triggers = nonemptyList(document.triggers, "triggers").map((trigger, index) =>
    parseTrigger(trigger, `triggers[${index}]`, stage),
  );
  validateTriggerIdentities(triggers);
  const runsOnTask = optionalBoolean(document.runs_on_task, "runs_on_task");
  if (runsOnTask === true && triggers.some(({ use }) => use === "schedule")) {
    problem("runs_on_task", "A schedule runs with no task.");
  }
  if (
    runsOnTask === false &&
    triggers.some(
      ({ use }) => use === "task_enters_stage" || use === "task_leaves_stage" || use === "pull_request_updated",
    )
  ) {
    problem("runs_on_task", "This trigger runs on a task.");
  }
  const steps = document.steps;
  if (!Array.isArray(steps)) problem("steps", "Enter a list of steps.");
  return {
    apiVersion: "v2",
    kind: "Automation",
    name,
    icon,
    timeout_seconds: timeout,
    runs_on_task: runsOnTask,
    skip_setup_commands: optionalBoolean(document.skip_setup_commands, "skip_setup_commands"),
    skip_repository_checkout: optionalBoolean(document.skip_repository_checkout, "skip_repository_checkout"),
    agent: document.agent === undefined ? undefined : parseAgent(document.agent),
    triggers,
    steps: steps.map(parseStep),
  };
}

function parseTrigger(value: unknown, path: string, stage: CustomAutomationColumn): AutomationTrigger {
  const trigger = mapping(value, path, ["id", "use", "config"]);
  const id = optionalText(trigger.id, `${path}.id`);
  switch (trigger.use) {
    case "schedule": {
      const config = mapping(trigger.config, `${path}.config`, ["cron", "utc_offset"]);
      const cron = singleLine(config.cron, `${path}.config.cron`);
      if (![5, 6].includes(cron.trim().split(/\s+/).length)) {
        problem(`${path}.config.cron`, "Enter a cron expression with five or six fields.");
      }
      const offset = config.utc_offset;
      if (
        offset !== undefined &&
        (typeof offset !== "number" || !Number.isFinite(offset) || offset < -12 || offset > 14)
      ) {
        problem(`${path}.config.utc_offset`, "Enter a UTC offset from -12 through 14.");
      }
      return { id, use: "schedule", config: { cron, utc_offset: offset } };
    }
    case "pull_request_updated":
      if (stage !== "verify") problem(`${path}.use`, "This trigger requires Verify.");
      break;
    case "task_enters_stage":
      if (stage === "verify") problem(`${path}.use`, "Use pull_request_updated in Verify.");
      break;
    case "manual":
    case "task_leaves_stage":
      break;
    default:
      problem(`${path}.use`, "Choose a supported trigger.");
  }
  if (trigger.config !== undefined) problem(`${path}.config`, "Only schedule accepts a configuration.");
  return { id, use: trigger.use };
}

function validateTriggerIdentities(triggers: AutomationTrigger[]) {
  const ids = new Set<string>();
  const events = new Set<string>();
  triggers.forEach(({ id, use }, index) => {
    const path = `triggers[${index}]`;
    if (use !== "schedule") {
      if (events.has(use)) problem(`${path}.use`, "Use this trigger once.");
      events.add(use);
    }
    if (id) {
      if (ids.has(id)) problem(`${path}.id`, "Use a unique trigger ID.");
      ids.add(id);
    }
  });
}

function parseAgent(value: unknown): AutomationAgent {
  const entry = mapping(value, "agent", ["harness", "model", "credentials"]);
  const harness = entry.harness;
  const model = optionalText(entry.model, "agent.model");
  if (harness === undefined && model !== undefined && entry.credentials === undefined) return { model };
  if (harness !== "superplane" && harness !== "claude-code" && harness !== "codex" && harness !== "openrouter") {
    problem("agent.harness", "Choose a supported agent.");
  }
  if (entry.credentials === undefined) return { harness, model };
  if (harness === "superplane") problem("agent.credentials", "SuperPlane uses workspace credentials.");
  const credentials = mapping(entry.credentials, "agent.credentials", ["source", "name"]);
  if (credentials.source === "integration") {
    return {
      harness,
      model,
      credentials: { source: "integration", name: singleLine(credentials.name, "agent.credentials.name") },
    };
  }
  if (credentials.source === "subscription") {
    if (harness !== "codex") problem("agent.credentials.source", "A ChatGPT subscription requires Codex.");
    if (credentials.name !== undefined) problem("agent.credentials.name", "Remove the subscription name.");
    return { harness, model, credentials: { source: "subscription" } };
  }
  if (credentials.source === "hosted") {
    if (credentials.name !== undefined) problem("agent.credentials.name", "Remove the hosted connection name.");
    return { harness, model, credentials: { source: "hosted" } };
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
    if (!fields.includes(field)) problem(`${path}.${field}`, "Remove the unknown field.");
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
        executionTimeoutSeconds: definition.timeout_seconds,
        skipSetupCommands: definition.skip_setup_commands,
        skipRepositoryCheckout: definition.skip_repository_checkout,
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
  if (definition.runs_on_task !== undefined) return definition.runs_on_task;
  if (definition.triggers.every(({ use }) => use !== "schedule")) return true;
  if (definition.triggers.every(({ use }) => use === "schedule")) return false;
  // Keep both task and schedule triggers when the YAML already combines them.
  return undefined;
}

function canvasTrigger(trigger: AutomationTrigger, index: number): CustomAutomationTrigger {
  return {
    id: trigger.id === undefined ? `implicit:${index}` : `public:${trigger.id}`,
    publicId: trigger.id,
    mode:
      trigger.use === "schedule"
        ? "scheduled"
        : trigger.use === "manual"
          ? "manual"
          : trigger.use === "task_leaves_stage"
            ? "leave"
            : "automatic",
    schedule:
      trigger.use === "schedule"
        ? { cronExpression: trigger.config.cron, utcOffset: trigger.config.utc_offset ?? 0 }
        : customAutomationSchedule(),
  };
}

export function automationDefinitionTriggers(
  triggers: CustomAutomationTrigger[],
  stage: CustomAutomationColumn,
): AutomationTrigger[] {
  return triggers.map(({ publicId, mode, schedule }) => {
    if (mode === "scheduled") {
      return {
        id: publicId,
        use: "schedule",
        config: { cron: schedule.cronExpression, utc_offset: schedule.utcOffset },
      };
    }
    if (mode === "manual") return { id: publicId, use: "manual" };
    if (mode === "leave") return { id: publicId, use: "task_leaves_stage" };
    return { id: publicId, use: stage === "verify" ? "pull_request_updated" : "task_enters_stage" };
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
