import type { ReactNode } from "react";

import type { RepositoryBranchPrompt } from "./repositoryBranchPrompt";
import { useOnboardingBranchPrompt } from "./useOnboardingBranchPrompt";

/** Loads branch names only after onboarding asks the user to choose one. */
export function RepositoryBranchPromptLoader(props: {
  organizationId: string;
  integrationId: string;
  repository: string | null;
  branch: string;
  onChange: (branch: string) => void;
  render: (prompt: RepositoryBranchPrompt | undefined) => ReactNode;
}) {
  const prompt = useOnboardingBranchPrompt({
    enabled: true,
    organizationId: props.organizationId,
    integrationId: props.integrationId,
    repository: props.repository,
    branch: props.branch,
    onChange: props.onChange,
  });
  return props.render(prompt);
}
