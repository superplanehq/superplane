export function bitbucketForgePhase(
  identityLinked: boolean,
  repositoryCount: number,
  installedWorkspaceCount: number,
): "connect" | "grant" | "choose" {
  if (!identityLinked) return "connect";
  if (repositoryCount === 0 && installedWorkspaceCount === 0) return "grant";
  return "choose";
}

export function selectWorkspaceRepository(
  model: {
    setup: { selectedRepo: string | null; selectRepo: (repository: string) => void };
    setChosenDefaultBranch: (branch: string) => void;
    clearDefaultBranchRequirement: () => void;
  },
  repository: string,
) {
  if (repository !== model.setup.selectedRepo) {
    model.setChosenDefaultBranch("");
    model.clearDefaultBranchRequirement();
  }
  model.setup.selectRepo(repository);
}
