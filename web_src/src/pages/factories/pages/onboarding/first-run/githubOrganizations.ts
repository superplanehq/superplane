function repositoryOwner(repository: string): string {
  return repository.split("/")[0]?.trim() ?? "";
}

function sameOrganization(left: string, right: string): boolean {
  return left.toLowerCase() === right.toLowerCase();
}

/** Distinct GitHub accounts and organizations that own the repositories, in name order. */
export function repositoryOrganizations(repositories: string[]): string[] {
  const organizations: string[] = [];
  for (const repository of repositories) {
    const owner = repositoryOwner(repository);
    if (!owner || organizations.some((organization) => sameOrganization(organization, owner))) continue;
    organizations.push(owner);
  }
  return organizations.sort((left, right) => left.localeCompare(right, undefined, { sensitivity: "base" }));
}

export function repositoriesInOrganization(repositories: string[], organization: string): string[] {
  return repositories.filter((repository) => repositoryBelongsTo(repository, organization));
}

/** The listed organization with this name, or null when the list does not have it. */
export function findOrganization(organizations: string[], name: string | null): string | null {
  if (!name) return null;
  return organizations.find((organization) => sameOrganization(organization, name)) ?? null;
}

export function repositoryBelongsTo(repository: string, organization: string): boolean {
  return sameOrganization(repositoryOwner(repository), organization);
}

export function ownerOfRepository(repository: string | null): string | null {
  if (!repository) return null;
  return repositoryOwner(repository) || null;
}
