import type { FieldOption, Predicate, RuntimeValue, TriggerDefinition } from "@superplanehq/sdk";
import { getHeader, parseRepositoryReference, type GitHubPushEvent } from "../lib/github.js";

export interface OnPushConfiguration {
  repository?: string;
  refs?: Predicate[];
}

const DEFAULT_REF_PREDICATES: Predicate[] = [
  {
    type: "equals",
    value: "refs/heads/main",
  },
];

const ALL_PREDICATE_OPERATORS: FieldOption[] = [
  {
    label: "Equals",
    value: "equals",
  },
  {
    label: "Not Equals",
    value: "notEquals",
  },
  {
    label: "Matches",
    value: "matches",
  },
];

export const onPush = {
  name: "github.onPush",
  integration: "github",
  label: "On Push",
  description: "Listen to GitHub push events",
  configuration: [
    {
      name: "repository",
      label: "Repository",
      type: "integration-resource",
      required: true,
      description: "Repository to monitor for push events",
      typeOptions: {
        resource: {
          type: "repository",
          useNameAsValue: true,
        },
      },
    },
    {
      name: "refs",
      label: "Refs",
      type: "any-predicate-list",
      required: true,
      description: "List of ref predicates used to decide whether the push event should trigger execution.",
      default: DEFAULT_REF_PREDICATES,
      typeOptions: {
        anyPredicateList: {
          operators: ALL_PREDICATE_OPERATORS,
        },
      },
    },
  ],
  async setup({ configuration, runtime }) {
    const config = normalizeConfiguration(configuration);
    await runtime.integration.requestWebhook({
      eventType: "push",
      repository: config.repository,
    });
  },
  async handleWebhook({ configuration, body, headers, runtime }) {
    await runtime.logger.info("Received GitHub push webhook", {
      trigger: "github.onPush",
    });

    const eventType = getHeader(headers, "X-GitHub-Event");
    if (!eventType) {
      throw new Error("missing X-GitHub-Event header");
    }

    if (eventType !== "push") {
      await runtime.logger.info("Ignoring GitHub webhook because event type is not push", {
        eventType,
      });
      return {
        status: 200,
      };
    }

    const payload = JSON.parse(new TextDecoder().decode(body)) as GitHubPushEvent;
    if (payload.deleted === true) {
      await runtime.logger.info("Ignoring GitHub webhook because it represents a branch deletion");
      return {
        status: 200,
      };
    }

    const config = normalizeConfiguration(configuration);
    const ref = typeof payload.ref === "string" ? payload.ref : "";
    if (!ref) {
      throw new Error("missing ref");
    }

    if (!matchesAnyPredicate(config.refs, ref)) {
      await runtime.logger.info("Ignoring GitHub push because the ref did not match the configured filters", {
        ref,
      });
      return {
        status: 200,
      };
    }

    await runtime.events.emit("github.push", payload);
    return {
      status: 200,
    };
  },
  async cleanup() {},
} satisfies TriggerDefinition<OnPushConfiguration>;

function normalizeConfiguration(configuration: OnPushConfiguration): { repository: string; refs: Predicate[] } {
  const raw = (configuration ?? {}) as OnPushConfiguration;
  const repository = parseRepositoryReference(raw.repository ?? null);
  const refs = normalizePredicates((raw.refs ?? DEFAULT_REF_PREDICATES) as unknown as RuntimeValue);
  return { repository, refs };
}

function normalizePredicates(value: RuntimeValue): Predicate[] {
  if (!Array.isArray(value)) {
    return defaultRefPredicates();
  }

  const predicates = value
    .map((entry) => normalizePredicate(entry))
    .filter((entry): entry is Predicate => entry !== null);

  return predicates.length > 0 ? predicates : defaultRefPredicates();
}

function normalizePredicate(value: RuntimeValue): Predicate | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const record = value as Record<string, RuntimeValue>;
  return {
    type: normalizePredicateType(record.type),
    value: typeof record.value === "string" ? record.value : "",
  };
}

function defaultRefPredicates(): Predicate[] {
  return [{ type: "equals", value: "refs/heads/main" }];
}

function matchesAnyPredicate(predicates: Predicate[], ref: string): boolean {
  return predicates.some((predicate) => matchesPredicate(predicate, ref));
}

function matchesPredicate(predicate: Predicate, ref: string): boolean {
  const value = predicate.value;
  switch (predicate.type) {
    case "notEquals":
      return ref !== value;
    case "matches":
      return safeMatch(value, ref);
    case "equals":
    default:
      return ref === value;
  }
}

function normalizePredicateType(value: RuntimeValue): Predicate["type"] {
  switch (value) {
    case "notEquals":
    case "matches":
      return value;
    case "equals":
    default:
      return "equals";
  }
}

function safeMatch(pattern: string, value: string): boolean {
  try {
    return new RegExp(pattern).test(value);
  } catch {
    return false;
  }
}
