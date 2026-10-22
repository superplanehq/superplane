import type { AutomationRun } from "@/ui/AutomationRunHistory/types";
import type { RunHistoryIntakeSource } from "./intakeRunData";

/** Recognize the recorded source item without making requests to its provider. */
export function intakeRunSource(
  sourceId: RunHistoryIntakeSource,
  payload: Record<string, unknown>,
): AutomationRun["source"] {
  switch (sourceId) {
    case "github-issues":
      return githubSource(payload, false);
    case "dependabot-alerts":
      return githubSource(payload, true);
    case "sentry-exceptions": {
      const issue = record(record(payload.data).issue ?? payload.issue);
      return source(text(issue.shortId, issue.id) || "Sentry issue", issue.title, issue.permalink);
    }
    case "linear-issues": {
      const issue = record(payload.data);
      return source(text(issue.identifier) || "Linear issue", issue.title, text(payload.url, issue.url));
    }
    case "jira-issues": {
      const issue = record(payload.issue);
      return source(text(issue.key) || "Jira issue", record(issue.fields).summary, payload.url);
    }
    case "productive-tasks":
      return productiveSource(payload);
    case "pagerduty-incidents": {
      const incident = record(payload.incident);
      return source(
        text(incident.incident_number, incident.id) || "PagerDuty incident",
        incident.title,
        incident.html_url,
      );
    }
    case "datadog":
      return datadogSource(payload);
    case "scheduled-intake":
      return { reference: "Schedule", title: "Scheduled review" };
  }
}

function productiveSource(payload: Record<string, unknown>): AutomationRun["source"] {
  const task = record(payload.data);
  return source(task.id ? `#${text(task.id)}` : "Productive task", record(task.attributes).title, payload.url);
}

function datadogSource(payload: Record<string, unknown>): AutomationRun["source"] {
  const id = text(payload.issue_id, payload.id, payload.alert_id);
  return source(
    id.length > 16 ? `${id.slice(0, 8)}…` : id || "Datadog alert",
    payload.title,
    payload.issue_url ?? payload.link,
  );
}

function githubSource(payload: Record<string, unknown>, dependabot: boolean): AutomationRun["source"] {
  const issue = record(dependabot ? payload.alert : payload.issue);
  const repository = record(payload.repository);
  const name = text(repository.name, repository.full_name).split("/").pop();
  const reference = `${name || (dependabot ? "Alert" : "Issue")}${issue.number ? `#${text(issue.number)}` : ""}`;
  const title = dependabot ? record(issue.security_advisory).summary : issue.title;
  return source(reference, title, issue.html_url);
}

function source(reference: string, title: unknown, url: unknown): AutomationRun["source"] {
  if (!text(title)) return undefined;
  const address = text(url);
  return { reference, title: text(title), url: /^https?:\/\//i.test(address) ? address : undefined };
}

export function record(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
}

export function text(...values: unknown[]): string {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) return value.trim();
    if (typeof value === "number" && Number.isFinite(value)) return String(value);
  }
  return "";
}
