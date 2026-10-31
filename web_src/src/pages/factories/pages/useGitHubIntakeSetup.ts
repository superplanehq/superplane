import { useIntakeOwnerDraft } from "./useIntakeOwnerDraft";
import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import { GITHUB_INTAKE_SETUP_COPY } from "./githubIntakeSetupCopy";

export function useGitHubIntakeSetup(organizationId: string, factoryId: string) {
  const [skipInitialImport, setSkipInitialImport] = useState(false);
  const [error, setError] = useState<string>();

  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const owner = useIntakeOwnerDraft(organizationId, factoryId);

  const createGithubIntake = async () => {
    setError(undefined);
    try {
      const ownerSettings = owner.settings();
      await createIntake.mutateAsync({
        ...(Object.keys(ownerSettings).length ? { settings: ownerSettings } : {}),
        source: "SOURCE_GITHUB_ISSUES",
        ...(skipInitialImport ? { skipInitialImport: true } : {}),
      });
      return true;
    } catch (cause) {
      setError(getApiErrorMessage(cause, GITHUB_INTAKE_SETUP_COPY.wizardCreateError));
      return false;
    }
  };

  return {
    owner,
    skipInitialImport,
    setSkipInitialImport,
    error,
    createIntake,
    createGithubIntake,
  };
}

export type GitHubIntakeSetupModel = ReturnType<typeof useGitHubIntakeSetup>;
