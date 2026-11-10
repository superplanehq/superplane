export type RepositoryBranchPrompt = {
  branch: string;
  choices: string[];
  hasListedBranches: boolean;
  onChange: (branch: string) => void;
};

/** A missing prompt means the branch is already known. */
export function repositoryBranchChosen(prompt: RepositoryBranchPrompt | undefined) {
  if (!prompt) return true;
  return prompt.branch.trim() !== "";
}
