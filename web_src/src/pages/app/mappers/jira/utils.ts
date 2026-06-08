import type { MetadataItem } from "@/ui/metadataList";
import type { JiraIssue, JiraProject } from "./types";

export function getIssueLabel(issue: JiraIssue | undefined): string {
  if (!issue) {
    return "";
  }
  const summary = issue.fields?.summary;
  if (issue.key && summary) {
    return `${issue.key} · ${summary}`;
  }
  return issue.key || summary || "";
}

export function getIssueUrl(issue: JiraIssue | undefined): string {
  if (!issue?.key || !issue.self) {
    return "";
  }

  try {
    const url = new URL(issue.self);
    return `${url.origin}/browse/${encodeURIComponent(issue.key)}`;
  } catch {
    return "";
  }
}

export function addDetail(details: Record<string, string>, label: string, value: string | undefined): void {
  if (value && value.trim() !== "") {
    details[label] = value;
  }
}

export function addProjectMetadata(
  metadata: MetadataItem[],
  project: JiraProject | undefined,
  configuredProject: string | undefined,
): void {
  const label = getProjectMetadataLabel(project, configuredProject);
  if (label) {
    metadata.push({ icon: "folder", label });
  }
}

export function addIssueKeyMetadata(
  metadata: MetadataItem[],
  icon: MetadataItem["icon"],
  issueKey: string | undefined,
) {
  if (issueKey && !issueKey.includes("{{")) {
    metadata.push({ icon, label: issueKey });
  }
}

function getProjectMetadataLabel(project: JiraProject | undefined, configuredProject: string | undefined) {
  if (project?.name || project?.key) {
    return project.name || project.key;
  }
  return configuredProject;
}
