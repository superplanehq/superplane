import type { FactoriesWorkOrderEvent } from "@/api-client";

export interface StepActionRow {
  id: string;
  title: string;
  detail?: string;
}

const OWNER_EVENT = "order.assignees.updated";
const CHECK_EVENT = "order.check.reported";

/** One timeline row per owner assignment or check report on this card's runs. Oldest first. */
export function stepActionRows(
  events: FactoriesWorkOrderEvent[],
  runIds: ReadonlySet<string>,
  resolveName: (userId: string) => string | undefined,
): StepActionRow[] {
  return events
    .map((event, index) => ({ event, index }))
    .filter(({ event }) => {
      const runId = readRunId(event.event);
      return runId != null && runIds.has(runId);
    })
    .sort((left, right) => {
      const byTime = (left.event.timestamp ?? "").localeCompare(right.event.timestamp ?? "");
      if (byTime !== 0) {
        return byTime;
      }
      return right.index - left.index;
    })
    .flatMap(({ event, index }) => {
      const row = stepActionRow(event, index, resolveName);
      return row ? [row] : [];
    });
}

function stepActionRow(
  event: FactoriesWorkOrderEvent,
  index: number,
  resolveName: (userId: string) => string | undefined,
): StepActionRow | undefined {
  const payload = record(event.event);
  if (!payload) {
    return undefined;
  }
  if (event.type === OWNER_EVENT) {
    return ownerRow(event, index, payload, resolveName);
  }
  if (event.type === CHECK_EVENT) {
    return checkRow(event, index, payload);
  }
  return undefined;
}

function ownerRow(
  event: FactoriesWorkOrderEvent,
  index: number,
  payload: Record<string, unknown>,
  resolveName: (userId: string) => string | undefined,
): StepActionRow {
  const assigned = Array.isArray(payload.assigned) ? payload.assigned : [];
  const userId = readId(assigned[0]);
  const name = (userId && resolveName(userId)) || "Unknown member";
  const reason = text(payload.reason);
  return {
    id: actionId(event, index),
    title: `Assigned ${name} as owner`,
    detail: reason || undefined,
  };
}

function checkRow(
  event: FactoriesWorkOrderEvent,
  index: number,
  payload: Record<string, unknown>,
): StepActionRow | undefined {
  const check = record(payload.check);
  if (!check) {
    return undefined;
  }
  const name = text(check.name) || text(check.key) || "check";
  const score = checkScoreText(check);
  const summary = text(check.summary);
  return {
    id: actionId(event, index),
    title: score ? `Reported ${name} ${score}` : `Reported ${name}`,
    detail: summary || undefined,
  };
}

function actionId(event: FactoriesWorkOrderEvent, index: number): string {
  return `${event.type ?? "action"}-${event.timestamp ?? "none"}-${index}`;
}

function checkScoreText(check: Record<string, unknown>): string {
  if (typeof check.score !== "number") {
    return "";
  }
  if (check.format === "boolean") {
    return check.score > 0 ? "Pass" : "Fail";
  }
  if (check.format === "percent") {
    return `${check.score}%`;
  }
  if (typeof check.maxScore === "number" && check.maxScore > 0) {
    return `${check.score}/${check.maxScore}`;
  }
  return String(check.score);
}

function readRunId(payload: FactoriesWorkOrderEvent["event"]): string | undefined {
  const run = record(record(payload)?.run);
  return text(run?.id) || undefined;
}

function readId(value: unknown): string | undefined {
  return text(record(value)?.id) || undefined;
}

function record(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}

function text(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
