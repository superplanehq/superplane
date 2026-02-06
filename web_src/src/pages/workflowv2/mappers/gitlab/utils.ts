import { formatTimeAgo } from "@/utils/date";
import { ExecutionInfo } from "../types";

export function buildGitlabSubtitle(content: string | undefined, createdAt?: string): string {
  const trimmed = (content || "").trim();
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";

  if (trimmed && timeAgo) {
    return `${trimmed} · ${timeAgo}`;
  }
  return trimmed || timeAgo;
}

export function buildGitlabExecutionSubtitle(execution: ExecutionInfo, content?: string): string {
  const timestamp = execution.updatedAt || execution.createdAt;
  return buildGitlabSubtitle(content || "", timestamp);
}
