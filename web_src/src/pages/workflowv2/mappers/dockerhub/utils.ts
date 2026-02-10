import { MetadataItem } from "@/ui/metadataList";
import { DockerHubRepositoryConfiguration, DockerHubRepositoryMetadata } from "./types";

export function getRepositoryLabel(
  metadata?: DockerHubRepositoryMetadata,
  configuration?: DockerHubRepositoryConfiguration,
  fallback?: string,
): string | undefined {
  const repoName = metadata?.repository?.name || configuration?.repository || fallback;
  if (!repoName) {
    return undefined;
  }

  const namespace =
    metadata?.repository?.namespace || metadata?.namespace || configuration?.namespace || undefined;

  return namespace ? `${namespace}/${repoName}` : repoName;
}

export function buildRepositoryMetadataItems(
  metadata?: DockerHubRepositoryMetadata,
  configuration?: DockerHubRepositoryConfiguration,
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
