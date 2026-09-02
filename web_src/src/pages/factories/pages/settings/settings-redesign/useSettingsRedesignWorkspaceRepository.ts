import { useEffect, useMemo, useState } from "react";

import { usePermissions } from "@/contexts/usePermissions";
import { resolveGithubDefaultBranch, useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { useFactoryRepository } from "../useFactoryRepository";

export function useSettingsRedesignWorkspaceRepository() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const integrationId = factory.onboarding?.vcsIntegrationId ?? "";
  const resources = useIntegrationResources(organizationId, integrationId, "repository");
  const repositories = useMemo(() => namedRepositories(resources.data), [resources.data]);
  const savedRepository = factory.onboarding?.appRepository ?? "";
  const [repository, setRepository] = useState(savedRepository);
  const [isResolvingDefaultBranch, setIsResolvingDefaultBranch] = useState(false);
  const updateRepository = useFactoryRepository(organizationId, factoryId);
  const canUpdate = canAct("factories", "update");
  const isSaving = isResolvingDefaultBranch || updateRepository.isPending;
  const canSave =
    Boolean(repository) && repository !== savedRepository && canUpdate && !permissionsLoading && !isSaving;

  useEffect(() => {
    setRepository(savedRepository);
  }, [savedRepository]);

  return {
    factoryName: factory.name ?? "Workspace",
    integrationId,
    repositories,
    repository,
    setRepository,
    resourcesLoading: resources.isLoading,
    resourcesError: resources.isError,
    canUpdate,
    permissionsLoading,
    isSaving,
    canSave,
    save: () =>
      void persistWorkspaceRepository({
        organizationId,
        integrationId,
        repository,
        isSaving,
        mutate: (body) => updateRepository.mutateAsync(body),
        setResolving: setIsResolvingDefaultBranch,
      }),
  };
}

function namedRepositories(resources: Array<{ name?: string; id?: string }> | undefined) {
  return (resources ?? [])
    .map((resource) => resource.name ?? resource.id ?? "")
    .filter((repository): repository is string => Boolean(repository));
}

async function persistWorkspaceRepository({
  organizationId,
  integrationId,
  repository,
  isSaving,
  mutate,
  setResolving,
}: {
  organizationId: string;
  integrationId: string;
  repository: string;
  isSaving: boolean;
  mutate: (body: { repository: string; defaultBranch: string }) => Promise<unknown>;
  setResolving: (value: boolean) => void;
}) {
  if (!repository || !integrationId || isSaving) return;
  setResolving(true);
  try {
    const defaultBranch = await resolveGithubDefaultBranch(organizationId, integrationId, repository);
    await mutate({ repository, defaultBranch });
    showSuccessToast("Workspace repository updated.");
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to update workspace repository"));
  } finally {
    setResolving(false);
  }
}
