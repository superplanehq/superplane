import { MetadataItem } from "@/ui/metadataList";

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
