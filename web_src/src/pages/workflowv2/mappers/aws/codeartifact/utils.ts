import { MetadataItem } from "@/ui/metadataList";
import {
  CodeArtifactPackageVersionConfiguration,
  CodeArtifactPackageVersionDetail,
  CodeArtifactTriggerConfiguration,
  CodeArtifactTriggerMetadata,
} from "./types";

function resolveFilters(
  metadata?: CodeArtifactTriggerMetadata,
  configuration?: CodeArtifactTriggerConfiguration,
): CodeArtifactTriggerConfiguration {
  return metadata?.filters ?? configuration ?? {};
}

export function formatPackageName(namespace?: string | null, name?: string): string | undefined {
  if (!name) {
    return undefined;
  }

  if (!namespace) {
    return name;
  }

  return `${namespace}/${name}`;
}

export function formatPackageLabel(
  metadata?: CodeArtifactTriggerMetadata,
  configuration?: CodeArtifactTriggerConfiguration,
  detail?: CodeArtifactPackageVersionDetail,
): string | undefined {
  const filters = resolveFilters(metadata, configuration);
  const packageName = formatPackageName(
    filters.packageNamespace ?? detail?.packageNamespace,
    filters.packageName ?? detail?.packageName,
  );
  if (!packageName) {
    return undefined;
  }

  const version = filters.packageVersion ?? detail?.packageVersion;
  if (!version) {
    return packageName;
  }

  return `${packageName}@${version}`;
}

export function buildCodeArtifactMetadataItems(
  metadata?: CodeArtifactTriggerMetadata,
  configuration?: CodeArtifactTriggerConfiguration,
): MetadataItem[] {
  const filters = resolveFilters(metadata, configuration);
  const items: MetadataItem[] = [];

  if (filters.domainName) {
    items.push({
      icon: "database",
      label: filters.domainName,
    });
  }

  if (filters.repositoryName) {
    items.push({
      icon: "package",
      label: filters.repositoryName,
    });
  }

  const packageLabel = formatPackageLabel(metadata, configuration);
  if (packageLabel) {
    items.push({
      icon: "tag",
      label: packageLabel,
    });
  }

  return items;
}

export function buildCodeArtifactPackageMetadataItems(
  configuration?: CodeArtifactPackageVersionConfiguration,
): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (configuration?.domain) {
    items.push({
      icon: "database",
      label: configuration.domain,
    });
  }

  if (configuration?.repository) {
    items.push({
      icon: "package",
      label: configuration.repository,
    });
  }

  const packageName = formatPackageName(configuration?.namespace, configuration?.package);
  if (packageName) {
    const label = configuration?.version ? `${packageName}@${configuration.version}` : packageName;
    items.push({
      icon: "tag",
      label,
    });
  }

  return items;
}
