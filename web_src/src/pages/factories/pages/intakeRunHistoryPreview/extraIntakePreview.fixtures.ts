import type { LineIntakeSourceId } from "../lineIntakeModel";
import type { IntakeRunPreview } from "./intakeRunPreviewModel";

type ExtraSource = Exclude<LineIntakeSourceId, "github-issues" | "sentry-exceptions" | "dependabot-alerts">;
type SourceItem = NonNullable<IntakeRunPreview["source"]>;

const SOURCES: Record<ExtraSource, { prefix: string; offset: number; url: string; event: string }> = {
  "linear-issues": {
    prefix: "ENG-",
    offset: 241,
    url: "https://linear.app/example/issue/",
    event: "Issue added to project",
  },
  "jira-issues": {
    prefix: "PLAT-",
    offset: 1032,
    url: "https://example.atlassian.net/browse/",
    event: "Issue created",
  },
  "pagerduty-incidents": {
    prefix: "INC-",
    offset: 824,
    url: "https://example.pagerduty.com/incidents/",
    event: "Incident triggered",
  },
  "productive-tasks": {
    prefix: "#",
    offset: 563,
    url: "https://app.productive.io/1-example/tasks/",
    event: "Task created",
  },
  datadog: {
    prefix: "ERR-",
    offset: 4820,
    url: "https://app.datadoghq.com/error-tracking/issue/",
    event: "Error alert triggered",
  },
};
const INCIDENT_TITLES = [
  "API error rate is above the alert threshold",
  "Background jobs stopped processing",
  "Database connection pool is exhausted",
  "Checkout requests are timing out",
];
const ERROR_TITLES = [
  "TypeError in the checkout service",
  "Unhandled exception while processing a payment",
  "Null response while loading project settings",
  "Timeout while calling the billing API",
];

export function extraIntakePreview(sourceId: ExtraSource, index: number, title: string) {
  const spec = SOURCES[sourceId];
  const reference =
    sourceId === "datadog" ? `${datadogIssueId(index).slice(0, 8)}…` : `${spec.prefix}${spec.offset + index}`;
  const source = {
    reference,
    title:
      sourceId === "pagerduty-incidents"
        ? INCIDENT_TITLES[index % INCIDENT_TITLES.length]
        : sourceId === "datadog"
          ? ERROR_TITLES[index % ERROR_TITLES.length]
          : title,
    url: `${spec.url}${sourceId === "datadog" ? datadogIssueId(index) : sourceId === "productive-tasks" ? spec.offset + index : reference}`,
  };
  return { source, event: spec.event, payload: extraPayload(sourceId, source, index) };
}

function extraPayload(sourceId: ExtraSource, source: SourceItem, index: number): Record<string, unknown> {
  switch (sourceId) {
    case "linear-issues":
      return {
        type: "linear.issue",
        data: {
          action: "create",
          type: "Issue",
          data: {
            id: `linear-issue-${index}`,
            identifier: source.reference,
            title: source.title,
            url: source.url,
            description: "Keep the existing behavior for other projects.",
            project: { id: "project-1", name: "Platform" },
            labels: [],
          },
        },
      };
    case "jira-issues":
      return {
        type: "jira.issue",
        data: {
          action: "created",
          url: source.url,
          description: "Keep the existing behavior for other projects.",
          issue: {
            id: String(10000 + index),
            key: source.reference,
            fields: { summary: source.title, project: { key: "PLAT" }, status: { name: "To Do" }, labels: [] },
          },
        },
      };
    case "pagerduty-incidents":
      return {
        type: "pagerduty.incident.triggered",
        data: {
          incident: {
            id: source.reference,
            title: source.title,
            html_url: source.url,
            urgency: "high",
            status: "triggered",
            service: { summary: "Production API" },
          },
        },
      };
    case "productive-tasks":
      return {
        type: "productive.task",
        data: {
          event: "task.created",
          data: {
            id: String(563 + index),
            type: "tasks",
            attributes: { title: source.title, description: "Keep the existing behavior for other projects." },
            relationships: { task_list: { data: { id: "platform", type: "task_lists" } } },
          },
        },
      };
    case "datadog":
      return {
        type: "datadog.errorTrackingAlert",
        data: {
          title: source.title,
          description: "A new error was reported in production.",
          service: "checkout-api",
          env: "production",
          issue_id: datadogIssueId(index),
          issue_url: source.url,
          alert_transition: "Triggered",
        },
      };
  }
}

function datadogIssueId(index: number) {
  return `${(0x42b67c1e + index).toString(16)}-d8d6-43c7-93ef-1df5f30c6218`;
}
