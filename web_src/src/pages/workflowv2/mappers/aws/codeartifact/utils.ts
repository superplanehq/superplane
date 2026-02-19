import { PackageVersionDetail } from "./types";

export function formatPackageName(namespace?: string | null, name?: string): string | undefined {
  if (!name) {
    return undefined;
  }

  if (!namespace) {
    return name;
  }

  return `${namespace}/${name}`;
}

export function formatPackageLabel(detail?: PackageVersionDetail): string | undefined {
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
