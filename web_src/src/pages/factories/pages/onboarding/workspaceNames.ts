const PLACEHOLDER_BASE = "New workspace";
const MAX_PLACEHOLDER_ATTEMPTS = 500;

/**
 * Workspace names are unique inside an organization, so the name the wizard
 * creates the workspace with must avoid the names already in use. The user
 * sets the final name after they pick a repository; until then SuperPlane
 * keeps a placeholder.
 */
export function placeholderWorkspaceName(existingNames: string[]): string {
  const taken = new Set(existingNames.map((name) => name.trim().toLowerCase()));
  if (!taken.has(PLACEHOLDER_BASE.toLowerCase())) {
    return PLACEHOLDER_BASE;
  }

  for (let suffix = 2; suffix <= MAX_PLACEHOLDER_ATTEMPTS; suffix += 1) {
    const candidate = `${PLACEHOLDER_BASE} ${suffix}`;
    if (!taken.has(candidate.toLowerCase())) {
      return candidate;
    }
  }

  return `${PLACEHOLDER_BASE} ${Date.now()}`;
}

/** True for the temporary name used before the user confirms the real one. */
export function isPlaceholderWorkspaceName(name: string): boolean {
  const normalized = name.trim().toLowerCase();
  if (normalized === PLACEHOLDER_BASE.toLowerCase()) return true;
  return new RegExp(`^${PLACEHOLDER_BASE.toLowerCase()} \\d+$`).test(normalized);
}

/** "acme/payments-service" becomes "Payments Service". */
export function workspaceNameFromRepository(repository: string): string {
  const repositoryName = repository.split("/").pop() ?? "";
  const words = repositoryName.split(/[-_.\s]+/).filter(Boolean);
  return words.map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join(" ");
}
