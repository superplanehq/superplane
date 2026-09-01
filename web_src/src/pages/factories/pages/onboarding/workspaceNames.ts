export const PLACEHOLDER_WORKSPACE_NAME = "New workspace";

const MAX_NAME_ATTEMPTS = 500;

/**
 * Workspace names are unique inside an organization, so a name that is already
 * in use gets a counted suffix: "Payments Service", "Payments Service 2", …
 */
export function uniqueWorkspaceName(base: string, existingNames: Iterable<string>): string {
  const name = base.trim();
  const taken = new Set([...existingNames].map((existing) => existing.trim().toLowerCase()));
  if (!taken.has(name.toLowerCase())) {
    return name;
  }

  for (let suffix = 2; suffix <= MAX_NAME_ATTEMPTS; suffix += 1) {
    const candidate = `${name} ${suffix}`;
    if (!taken.has(candidate.toLowerCase())) {
      return candidate;
    }
  }

  return `${name} ${Date.now()}`;
}

/** True for the temporary name used before the user confirms the real one. */
export function isPlaceholderWorkspaceName(name: string): boolean {
  const normalized = name.trim().toLowerCase();
  if (normalized === PLACEHOLDER_WORKSPACE_NAME.toLowerCase()) return true;
  return new RegExp(`^${PLACEHOLDER_WORKSPACE_NAME.toLowerCase()} \\d+$`).test(normalized);
}

/** "acme/payments-service" becomes "Payments Service". */
export function workspaceNameFromRepository(repository: string): string {
  const repositoryName = repository.split("/").pop() ?? "";
  const words = repositoryName.split(/[-_.\s]+/).filter(Boolean);
  return words.map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join(" ");
}
