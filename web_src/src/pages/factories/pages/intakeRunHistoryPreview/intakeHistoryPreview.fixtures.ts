import type { LineIntakeSourceId } from "../lineIntakeModel";
import { extraIntakePreview } from "./extraIntakePreview.fixtures";
import { scheduledHistoryPreviewRuns } from "./scheduledHistoryPreview.fixtures";
import type { IntakePreviewSource, IntakeRunOutcome, IntakeRunPreview } from "./intakeRunPreviewModel";

const TITLES = [
  "Show the remaining credit balance after renewal",
  "Let users choose a model when creating a task",
  "Keep issue labels when reconnecting GitHub",
  "Add a compact list of recent imports",
  "Retry task creation after a temporary connection error",
  "Keep the source issue attached after moving a task",
  "Restore the selected repository when reopening settings",
  "Handle issue titles with long repository names and repeated event updates without expanding the height of a compact history row",
];
const SENTRY_TITLES = [
  "TypeError: Cannot read properties of undefined",
  "Database connection timed out during task creation",
  "Request failed with status code 502",
  "ChunkLoadError: Loading the settings page failed",
  "Permission denied when writing an attachment",
  "RangeError: Invalid time value",
  "WebSocket connection closed unexpectedly",
  "Unhandled promise rejection while loading the organization configuration after reconnecting an expired integration",
];
const DEPENDABOT_TITLES = [
  "Path traversal in a development dependency",
  "Cross-site scripting in markdown rendering",
  "Denial of service in the HTTP request parser",
  "Prototype pollution in a configuration utility",
  "Improper certificate validation in an API client",
  "Information exposure through debug output",
  "Uncontrolled resource consumption in archive extraction",
  "Regular expression denial of service when parsing specially crafted input with repeated nested groups",
];
const OUTCOMES: IntakeRunOutcome[] = [
  "created",
  "filtered",
  "filtered",
  "processing",
  "failed",
  "unknown",
  "cancelled",
  "filtered",
];
const SUMMARY: Record<IntakeRunOutcome, string> = {
  "no-proposal": "The scheduled review finished without proposing a task.",
  created: "SuperPlane created a task from this source event.",
  filtered: "This source event did not match the intake filters. No task was created.",
  processing: "SuperPlane is processing this source event. The final outcome is not available yet.",
  failed: "Processing failed before SuperPlane could create a task.",
  unknown: "This run completed, but the recorded data does not explain its intake outcome.",
  cancelled: "This run was cancelled before intake finished. No final outcome was recorded.",
};

function sourceItem(sourceId: IntakePreviewSource, index: number, title: string) {
  if (sourceId === "sentry-exceptions")
    return {
      reference: `SP-${8616 + index}`,
      title: SENTRY_TITLES[index],
      url: `https://sentry.io/organizations/example/issues/${8616 + index}/`,
    };
  if (sourceId === "dependabot-alerts")
    return {
      reference: `superplane#${128 + index}`,
      title: DEPENDABOT_TITLES[index],
      url: `https://github.com/superplanehq/superplane/security/dependabot/${128 + index}`,
    };
  return {
    reference: `superplane#${8711 + index}`,
    title,
    url: `https://github.com/superplanehq/superplane/issues/${8711 + index}`,
  };
}

function sourcePayload(sourceId: IntakePreviewSource, source: ReturnType<typeof sourceItem>, index: number) {
  const issue = {
    number: 8711 + index,
    title: source.title,
    html_url: source.url,
    state: "open",
    labels: [{ name: index === 0 ? "superplane" : "needs-review" }],
  };
  if (sourceId === "sentry-exceptions")
    return {
      type: "sentry.issue",
      data: {
        action: "created",
        issue: {
          id: String(8616 + index),
          title: source.title,
          permalink: source.url,
          level: "error",
          project: { slug: "superplane" },
        },
      },
    };
  if (sourceId === "dependabot-alerts")
    return {
      type: "github.dependabot.alert",
      data: {
        action: "created",
        alert: {
          number: 128 + index,
          html_url: source.url,
          security_advisory: { summary: source.title, severity: "moderate" },
        },
        repository: { full_name: "superplanehq/superplane" },
      },
    };
  return {
    type: "github.issue",
    data: {
      action: index === 0 ? "opened" : "labeled",
      issue,
      ...(index === 0 ? {} : { label: { name: "needs-review" } }),
      repository: { full_name: "superplanehq/superplane" },
    },
  };
}

function activityFor(
  outcome: IntakeRunOutcome,
  startedAt: string,
  payload: Record<string, unknown>,
  taskId: string,
): IntakeRunPreview["activities"] {
  const source: IntakeRunPreview["activities"][number] = {
    id: "source-event",
    name: "Source event",
    status: "Passed",
    recordedAt: startedAt,
    logs: { state: "empty" },
    outputs: payload,
    defaultTab: "outputs",
    outputsLabel: "Event data",
  };
  if (outcome === "unknown" || outcome === "no-proposal") return [source];
  const filter: IntakeRunPreview["activities"][number] = {
    id: "intake-filters",
    name: "Matches filters?",
    status: "Passed",
    recordedAt: startedAt,
    logs: { state: "empty" },
    outputs: outcome === "filtered" ? { false: [{}] } : { true: [{}] },
    defaultTab: "outputs",
  };
  if (outcome === "filtered") return [source, filter];
  return [
    source,
    filter,
    {
      id: "create-task",
      name: "Create task",
      status:
        outcome === "failed"
          ? "Failed"
          : outcome === "processing"
            ? "Running"
            : outcome === "cancelled"
              ? "Cancelled"
              : "Passed",
      recordedAt: startedAt,
      logs: { state: "empty" },
      outputs: outcome === "created" ? { default: [{ taskId }] } : undefined,
      error: outcome === "failed" ? "The task could not be saved because of a temporary connection error." : undefined,
      defaultTab: "outputs",
    },
  ];
}

function sourcePreview(sourceId: LineIntakeSourceId, index: number) {
  if (sourceId !== "github-issues" && sourceId !== "sentry-exceptions" && sourceId !== "dependabot-alerts") {
    return extraIntakePreview(sourceId, index, TITLES[index]);
  }
  const source = sourceItem(sourceId, index, TITLES[index]);
  const event =
    sourceId === "github-issues"
      ? index === 0
        ? "Issue opened"
        : "Label added"
      : sourceId === "sentry-exceptions"
        ? "Issue created"
        : "Alert created";
  return { source, event, payload: sourcePayload(sourceId, source, index) };
}

const EXECUTION_STATUS = {
  created: "Passed",
  filtered: "Passed",
  processing: "Running",
  failed: "Failed",
  unknown: "Passed",
  cancelled: "Cancelled",
  "no-proposal": "Passed",
} as const;

export function intakeHistoryPreviewRuns(sourceId: IntakePreviewSource): IntakeRunPreview[] {
  if (sourceId === "scheduled-intake") return scheduledHistoryPreviewRuns();
  return OUTCOMES.map((sampleOutcome, index) => {
    const outcome = sourceId === "pagerduty-incidents" && sampleOutcome === "filtered" ? "created" : sampleOutcome;
    const { source, event, payload } = sourcePreview(sourceId, index);
    const started = new Date(Date.now() - (index * 9 + 1) * 60_000);
    const startedAt = started.toISOString();
    return {
      id: `${(0xf5c428cf - index * 0x1234567).toString(16).padStart(8, "0")}-a7fa-43ef-bd63-5c1b569fc5b1`,
      source,
      event,
      outcome,
      summary: SUMMARY[outcome],
      startedAt,
      finishedAt:
        outcome === "processing"
          ? undefined
          : new Date(started.getTime() + (index % 2 === 0 ? 1000 : 500)).toISOString(),
      duration: outcome === "processing" ? `${index * 9 + 1}m` : index % 2 === 0 ? "1s" : "< 1s",
      executionStatus: EXECUTION_STATUS[outcome],
      task: outcome === "created" ? { id: `SUPER-${987 + index}`, title: source.title } : undefined,
      error: outcome === "failed" ? "The task could not be saved because of a temporary connection error." : undefined,
      activities: activityFor(outcome, startedAt, payload, `task-${987 + index}`).filter(
        (activity) => sourceId !== "pagerduty-incidents" || activity.id !== "intake-filters",
      ),
    };
  });
}
