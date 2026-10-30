import type { FirstRunScreen } from "./firstRunScreens";
import type { FirstRunBlocking } from "./useFirstRunBlockingAction";
import type { GitHubConnectionState } from "./useFirstRunGitHub";

type RepositorySaveModel = {
  defaultBranchRequired: boolean;
  chosenDefaultBranch: string;
  saveChosenDefaultBranch: (branch: string) => Promise<boolean>;
  clearDefaultBranchRequirement: () => void;
  requireDefaultBranch: () => void;
  selectBitbucketForgeRepository: (repository: string) => Promise<boolean | "needs-branch">;
  selectBitbucketRepository: (repository: string) => Promise<boolean | "needs-branch">;
  selectGitHubEnterpriseRepository: (repository: string) => Promise<boolean | "needs-branch">;
  selectCatalogRepository: (repository: {
    repositoryId?: string;
    fullName?: string;
  }) => Promise<boolean | "needs-branch">;
  setup: {
    selectedRepo: string | null;
    vcsHost: string | null;
    commitRepoStep: () => void;
  };
};

function repositorySaveResult(value: boolean | "needs-branch" | undefined): "failed" | "ready" | "needs-branch" {
  if (value === "needs-branch") return "needs-branch";
  return value ? "ready" : "failed";
}

export function saveSelectedRepository(args: {
  model: RepositorySaveModel;
  connection: GitHubConnectionState;
  forgeConfigured: boolean;
  githubEnterprise: boolean;
}): Promise<boolean | "needs-branch"> {
  const selectedRepo = args.model.setup.selectedRepo;
  if (!selectedRepo) return Promise.resolve(false);
  if (args.model.setup.vcsHost === "bitbucket") {
    args.model.setup.commitRepoStep();
    if (args.forgeConfigured) return args.model.selectBitbucketForgeRepository(selectedRepo);
    return args.model.selectBitbucketRepository(selectedRepo);
  }
  if (args.githubEnterprise) {
    args.model.setup.commitRepoStep();
    return args.model.selectGitHubEnterpriseRepository(selectedRepo);
  }
  const repository = args.connection.repositories.find((candidate) => candidate.fullName === selectedRepo);
  if (!repository) return Promise.resolve(false);
  args.model.setup.commitRepoStep();
  return args.model.selectCatalogRepository(repository);
}

export function continueFromRepositoryAction(args: {
  model: RepositorySaveModel;
  blocking: FirstRunBlocking;
  navigation: { goToScreen: (screen: FirstRunScreen) => void };
  saveRepository: () => Promise<boolean | "needs-branch">;
}) {
  return () =>
    args.blocking.run("saving-repository", async () => {
      if (args.model.defaultBranchRequired) {
        if (!args.model.chosenDefaultBranch.trim()) return;
        if (!(await args.model.saveChosenDefaultBranch(args.model.chosenDefaultBranch))) return;
        args.model.clearDefaultBranchRequirement();
        args.navigation.goToScreen("tickets");
        return;
      }
      const result = repositorySaveResult(await args.saveRepository());
      if (result === "ready") {
        args.model.clearDefaultBranchRequirement();
        args.navigation.goToScreen("tickets");
        return;
      }
      if (result === "needs-branch") args.model.requireDefaultBranch();
    });
}
