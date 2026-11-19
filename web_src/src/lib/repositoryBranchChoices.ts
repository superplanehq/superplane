export function branchNames(resources: { name?: string | null; id?: string | null }[] | undefined) {
  return (resources ?? []).map((resource) => resource.name?.trim() || resource.id?.trim() || "").filter(Boolean);
}

const preferredDefaultBranches = ["main", "master", "develop", "dev", "development", "trunk"];
const preferredDefaultBranchRank = new Map(preferredDefaultBranches.map((name, index) => [name, index]));

export function repositoryBranchChoices(listedBranchNames: string[], branch: string) {
  const names = new Set(listedBranchNames);
  if (branch) names.add(branch);
  return [...names].sort(compareBranchChoices);
}

function compareBranchChoices(left: string, right: string) {
  const leftRank = preferredDefaultBranchRank.get(left.toLowerCase()) ?? Number.POSITIVE_INFINITY;
  const rightRank = preferredDefaultBranchRank.get(right.toLowerCase()) ?? Number.POSITIVE_INFINITY;
  if (leftRank !== rightRank) return leftRank - rightRank;
  return left.localeCompare(right);
}
