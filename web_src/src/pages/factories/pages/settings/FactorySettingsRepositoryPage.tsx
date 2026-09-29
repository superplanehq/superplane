import { PermissionTooltip } from "@/components/PermissionGate";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { useIntegrationResources, resolveGithubDefaultBranch } from "@/hooks/useIntegrations";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { RepositoryPicker } from "../onboarding/onboardingSteps";
import { useGitHubOnboarding } from "../onboarding/useGitHubOnboarding";
import { type ReactNode, useEffect, useMemo, useState } from "react";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { useFactoryRepository } from "./useFactoryRepository";

export function FactorySettingsRepositoryPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const integrationId = factory.onboarding?.vcsIntegrationId ?? "";
  const resources = useIntegrationResources(organizationId, integrationId, "repository");
  const githubOnboarding = useGitHubOnboarding(organizationId, { poll: false });
  const catalogRepositories = githubOnboarding.data?.repositories;
  const repositories = useMemo(() => {
    const names = new Set(
      (catalogRepositories ?? [])
        .map((repository) => repository.fullName ?? "")
        .filter((repository): repository is string => Boolean(repository)),
    );
    for (const resource of resources.data ?? []) {
      const name = resource.name ?? resource.id ?? "";
      if (name) names.add(name);
    }
    return Array.from(names).sort((left, right) => left.localeCompare(right));
  }, [catalogRepositories, resources.data]);
  const [repository, setRepository] = useState(factory.onboarding?.appRepository ?? "");
  const [isResolvingDefaultBranch, setIsResolvingDefaultBranch] = useState(false);
  const updateRepository = useFactoryRepository(organizationId, factoryId);
  const canUpdate = canAct("factories", "update");

  usePageTitle(["Repository", "Settings", factory.name ?? "Workspace"]);

  useEffect(() => {
    setRepository(factory.onboarding?.appRepository ?? "");
  }, [factory.onboarding?.appRepository]);

  const isDirty = repository !== (factory.onboarding?.appRepository ?? "");
  const isSaving = isResolvingDefaultBranch || updateRepository.isPending;
  const save = async () => {
    if (!repository || !integrationId || isSaving) return;
    setIsResolvingDefaultBranch(true);
    try {
      const catalogRepository = catalogRepositories?.find((candidate) => candidate.fullName === repository);
      const defaultBranch =
        catalogRepository?.defaultBranch ??
        (await resolveGithubDefaultBranch(organizationId, integrationId, repository));
      await updateRepository.mutateAsync({ repository, defaultBranch });
      showSuccessToast("Workspace repository updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update workspace repository"));
    } finally {
      setIsResolvingDefaultBranch(false);
    }
  };

  let repositoryContent: ReactNode;
  if (!integrationId) {
    repositoryContent = (
      <p className="text-[13px] text-muted-foreground">
        Connect GitHub during workspace setup before you select a repository.
      </p>
    );
  } else if (resources.isLoading && githubOnboarding.isPending && repositories.length === 0) {
    repositoryContent = <p className="text-[13px] text-muted-foreground">Loading repositories...</p>;
  } else {
    repositoryContent = (
      <RepositoryPicker host="github" repos={repositories} selectedRepo={repository || null} onSelect={setRepository} />
    );
  }

  return (
    <FactorySettingsPageFrame
      title="Repository"
      subtitle="Select the GitHub repository for workspace work and issue intake."
    >
      <FactorySettingsCard title="GitHub repository" data-testid="factory-settings-repository">
        {repositoryContent}
        {resources.isError && githubOnboarding.isError && repositories.length === 0 ? (
          <p className="mt-3 text-[13px] text-destructive">We could not load repositories. Try again.</p>
        ) : null}
        <div className="mt-4 flex items-center justify-between gap-4 border-t border-border pt-4">
          <p className="text-[12px] text-muted-foreground">
            Saving updates factory GitHub issue intake and pull request automations.
          </p>
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You do not have permission to update this workspace."
          >
            <LoadingButton
              type="button"
              loading={isSaving}
              loadingText="Saving..."
              disabled={!repository || !isDirty || !canUpdate || permissionsLoading || isSaving}
              onClick={save}
              data-testid="factory-settings-repository-save"
            >
              Save repository
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}
