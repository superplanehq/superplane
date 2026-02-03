import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { CanvasesCanvasNodeExecution } from "@/api-client";

export type PredicateType = "equals" | "notEquals" | "matches";

export interface Predicate {
  type: PredicateType;
  value: string;
}

export function formatPredicate(predicate: Predicate): string {
  switch (predicate.type) {
    case "equals":
      return `=${predicate.value}`;
    case "notEquals":
      return `!=${predicate.value}`;
    case "matches":
      return `~${predicate.value}`;
    default:
      return predicate.value;
  }
}

export function createGithubMetadataItems(
  repositoryName: string | undefined,
  predicates: Predicate[] | undefined,
): MetadataItem[] {
  const metadataItems: MetadataItem[] = [];

  if (repositoryName) {
    metadataItems.push({
      icon: "book",
      label: repositoryName,
    });
  }

  if (predicates && predicates.length > 0) {
    metadataItems.push({
      icon: "funnel",
      label: predicates.map(formatPredicate).join(", "),
    });
  }

  return metadataItems;
}

export function buildGithubSubtitle(content: string | undefined, createdAt?: string): string {
  const trimmed = (content || "").trim();
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";

  if (trimmed && timeAgo) {
    return `${trimmed} · ${timeAgo}`;
  }
  return trimmed || timeAgo;
}

export function buildGithubExecutionSubtitle(execution: CanvasesCanvasNodeExecution, content?: string): string {
  const timestamp = execution.updatedAt || execution.createdAt;
  return buildGithubSubtitle(content || "", timestamp);
}
