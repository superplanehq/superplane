import { MetadataItem } from "@/ui/metadataList";
import { Predicate, formatPredicate } from "../../utils";
import {
  CodeArtifactPackageVersionConfiguration,
  CodeArtifactPackageVersionDetail,
  CodeArtifactTriggerConfiguration,
  CodeArtifactTriggerMetadata,
} from "./types";

export function formatPackageName(namespace?: string | null, name?: string): string | undefined {
  if (!name) {
    return undefined;
  }

  if (!namespace) {
    return name;
  }

  return `${namespace}/${name}`;
}

export function formatPackageLabel(detail?: CodeArtifactPackageVersionDetail): string | undefined {
  const packageName = formatPackageName(detail?.packageNamespace, detail?.packageName);
  if (!packageName) {
    return undefined;
  }

  const version = detail?.packageVersion;
  if (!version) {
    return packageName;
  }

  return `${packageName}@${version}`;
}

function formatPredicateList(predicates?: Predicate[]): string | undefined {
  if (!predicates || predicates.length === 0) {
    return undefined;
  }

  const formatted = predicates.map((predicate) => formatPredicate(predicate)).join(", ");
  return formatted.trim().length > 0 ? formatted : undefined;
}

export function buildCodeArtifactMetadataItems(
  metadata?: CodeArtifactTriggerMetadata,
  configuration?: CodeArtifactTriggerConfiguration,
): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (metadata?.repository?.domainName) {
    items.push({
      icon: "database",
      label: `Domain: ${metadata?.repository?.domainName}`,
    });
  }

  const repositoryLabel = metadata?.repository?.name || configuration?.repository;
  if (repositoryLabel) {
    items.push({
      icon: "boxes",
      label: `Repository: ${repositoryLabel}`,
    });
  }

  const packagesLabel = formatPredicateList(configuration?.packages);
  if (packagesLabel) {
    items.push({
      icon: "package",
      label: `Packages: ${packagesLabel}`,
    });
  }

  const versionsLabel = formatPredicateList(configuration?.versions);
  if (versionsLabel) {
    items.push({
      icon: "tag",
      label: `Versions: ${versionsLabel}`,
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
      icon: "boxes",
      label: configuration.repository,
    });
  }

  const packageName = formatPackageName(configuration?.namespace, configuration?.package);
  if (packageName) {
    const label = configuration?.version ? `${packageName}@${configuration.version}` : packageName;
    items.push({
      icon: "package",
      label,
    });
  }

  return items;
}
