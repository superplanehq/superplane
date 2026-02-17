import { MetadataItem } from "@/ui/metadataList";
import { Predicate, formatPredicate, buildSubtitle, buildExecutionSubtitle } from "../utils";

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

export const buildGithubSubtitle = buildSubtitle;
export const buildGithubExecutionSubtitle = buildExecutionSubtitle;
