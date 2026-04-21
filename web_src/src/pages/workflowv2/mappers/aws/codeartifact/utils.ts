export function formatPackageName(namespace?: string | null, name?: string): string | undefined {
  if (!name) {
    return undefined;
  }

  if (!namespace) {
    return name;
  }

  return `${namespace}/${name}`;
}
