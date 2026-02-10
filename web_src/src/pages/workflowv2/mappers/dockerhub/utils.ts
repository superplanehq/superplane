import { MetadataItem } from "@/ui/metadataList";
import { OnImagePushConfiguration, OnImagePushMetadata } from "./types";

export function getRepositoryLabel(
  metadata?: OnImagePushMetadata,
  configuration?: OnImagePushConfiguration,
): string | undefined {
  const repoName = metadata?.repository?.name || configuration?.repository;
  if (!repoName) {
    return undefined;
  }

  const namespace = metadata?.repository?.namespace;
  return namespace ? `${namespace}/${repoName}` : repoName;
}

export function buildRepositoryMetadataItems(
  metadata?: OnImagePushMetadata,
  configuration?: OnImagePushConfiguration,
): MetadataItem[] {
  const repositoryLabel = getRepositoryLabel(metadata, configuration);
  if (!repositoryLabel) {
    return [];
  }

  return [
    {
      icon: "package",
      label: repositoryLabel,
    },
  ];
}
