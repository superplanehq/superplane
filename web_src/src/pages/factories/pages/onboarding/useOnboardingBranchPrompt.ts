import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useMemo } from "react";

import type { RepositoryBranchPrompt } from "./repositoryBranchPrompt";
import { branchNames, repositoryBranchChoices } from "../settings/useFactoryRepositoryDraft";

/** Branch list for the repository step. Shown only when no branch is known. */
export function useOnboardingBranchPrompt(args: {
  enabled: boolean;
  organizationId: string;
  integrationId: string;
  repository: string | null;
  branch: string;
  onChange: (branch: string) => void;
}): RepositoryBranchPrompt | undefined {
  const resources = useIntegrationResources(
    args.organizationId,
    args.integrationId,
    "branch",
    args.repository ? { repository: args.repository } : undefined,
    { enabled: args.enabled && Boolean(args.repository && args.integrationId) },
  );
  const listedBranchNames = useMemo(() => branchNames(resources.data), [resources.data]);
  const choices = useMemo(
    () => repositoryBranchChoices(listedBranchNames, args.branch),
    [args.branch, listedBranchNames],
  );
  if (!args.enabled) return undefined;
  return {
    branch: args.branch,
    choices,
    hasListedBranches: listedBranchNames.length > 0,
    onChange: args.onChange,
  };
}
