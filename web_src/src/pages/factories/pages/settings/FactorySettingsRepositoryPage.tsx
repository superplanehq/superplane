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
  const bitbucket = factory.onboarding?.vcsProvider === "bitbucket";
  const resources = useIntegrationResources(organizationId, integrationId, "repository");
  const githubOnboarding = useGitHubOnboarding(organizationId, { poll: false });
  const usesCatalogRepository = Boolean(factory.onboarding?.appRepositoryId || factory.onboarding?.backlogRepositoryId);
  const catalogRepositories = githubOnboarding.data?.repositories;
  const repositories = useMemo(() => {
    const names = usesCatalogRepository
      ? (catalogRepositories ?? []).map((repository) => repository.fullName ?? "")
      : (resources.data ?? []).map((resource) => resource.name ?? resource.id ?? "");
    return names
      .filter((repository): repository is string => Boolean(repository))
      .sort((left, right) => left.localeCompare(right));
  }, [catalogRepositories, resources.data, usesCatalogRepository]);
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
      const catalogRepository = usesCatalogRepository
        ? catalogRepositories?.find((candidate) => candidate.fullName === repository)
        : undefined;
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
        {bitbucket
          ? "Connect Bitbucket during workspace setup before you select a repository."
          : "Connect GitHub during workspace setup before you select a repository."}
      </p>
    );
  } else if ((usesCatalogRepository ? githubOnboarding.isPending : resources.isLoading) && repositories.length === 0) {
    repositoryContent = <p className="text-[13px] text-muted-foreground">Loading repositories...</p>;
  } else {
    repositoryContent = (
      <RepositoryPicker
        host={bitbucket ? "bitbucket" : "github"}
        repos={repositories}
        selectedRepo={repository || null}
        onSelect={setRepository}
      />
    );
  }

  return (
    <FactorySettingsPageFrame
      title="Repository"
      subtitle={
        bitbucket
          ? "Select the Bitbucket repository for this workspace."
          : "Select the GitHub repository for workspace work and issue intake."
      }
    >
      <FactorySettingsCard
        title={bitbucket ? "Bitbucket repository" : "GitHub repository"}
        data-testid="factory-settings-repository"
      >
        {repositoryContent}
        {(usesCatalogRepository ? githubOnboarding.isError : resources.isError) && repositories.length === 0 ? (
          <p className="mt-3 text-[13px] text-destructive">We could not load repositories. Try again.</p>
        ) : null}
        <div className="mt-4 flex items-center justify-between gap-4 border-t border-border pt-4">
          <p className="text-[12px] text-muted-foreground">
            {bitbucket
              ? "Saving updates the repository this workspace uses."
              : "Saving updates factory GitHub issue intake and pull request automations."}
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
