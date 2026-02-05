import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { CanvasesCanvasNodeExecution } from "@/api-client";
import { Predicate, formatPredicate } from "../utils";

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
