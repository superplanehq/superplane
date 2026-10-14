import { useState } from "react";
import { X } from "lucide-react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  useCreateFactoryAgentResource,
  useDeleteFactoryAgentResource,
  useFactoryAgentResources,
} from "@/hooks/useFactoryAgentResources";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showInfoToast, showSuccessToast } from "@/lib/toast";

import { IntegrationChoiceIcon, RepositoryPicker } from "../onboarding/onboardingSteps";
import type { VcsHostId } from "../onboarding/onboardingFixtures";
import { FirstRunMissingAccessLine } from "../onboarding/first-run/FirstRunMissingAccessLine";
import { useBitbucketOnboarding } from "../onboarding/useBitbucketOnboarding";
import { useGitHubOnboarding } from "../onboarding/useGitHubOnboarding";
import { navigateGitHubWindow, openGitHubWindow } from "../onboarding/useFirstRunGitHub";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";
import { useWorkspaceRepositoryChoices } from "./useWorkspaceRepositoryChoices";

type RepositoryWorkspace = {
  onboarding?: {
    vcsIntegrationId?: string;
    vcsProvider?: string;
    appRepositoryId?: string;
    backlogRepositoryId?: string;
  } | null;
};

function contextRepoName(resource: { name?: string | null; repository?: string | null }): string {
  return resource.repository ?? resource.name ?? "";
}

function ContextRepoList({
  resources,
  host,
  canUpdate,
  isSaving,
  onRemove,
}: {
  resources: { id?: string | null; name?: string | null; repository?: string | null }[];
  host: VcsHostId;
  canUpdate: boolean;
  isSaving: boolean;
  onRemove: (resourceId: string) => void;
}) {
  return (
    <ul
      className="divide-y divide-border overflow-hidden rounded-lg border border-border"
      data-testid="agent-cross-repo-list"
    >
      {resources.map((resource) => {
        const name = contextRepoName(resource);
        return (
          <li key={resource.id ?? name} className="flex min-w-0 items-center gap-3 px-3 py-2.5">
            <IntegrationChoiceIcon name={host} />
            <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{name}</span>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-8 shrink-0 text-muted-foreground"
              disabled={!canUpdate || isSaving}
              aria-label={AGENT_RESOURCES_COPY.removeRepository}
              data-testid={`agent-cross-repo-remove-${resource.id}`}
              onClick={() => resource.id && onRemove(resource.id)}
            >
              <X className="size-4" strokeWidth={1.75} aria-hidden />
            </Button>
          </li>
        );
      })}
    </ul>
  );
}

export function CrossRepoContextCard({
  organizationId,
  factoryId,
  factory,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  factory: RepositoryWorkspace | null | undefined;
  canUpdate: boolean;
}) {
  const [adding, setAdding] = useState(false);
  const contextRepos = useFactoryAgentResources(organizationId, factoryId, "KIND_CONTEXT_REPO");
  const createResource = useCreateFactoryAgentResource(organizationId, factoryId);
  const deleteResource = useDeleteFactoryAgentResource(organizationId, factoryId);
  const choices = useWorkspaceRepositoryChoices(organizationId, factory);

  const added = (contextRepos.data ?? []).map(contextRepoName).filter(Boolean);
  const available = choices.repositories.filter((name) => !added.includes(name));
  const isSaving = createResource.isPending || deleteResource.isPending;

  const choose = async (name: string) => {
    const repository = name.trim();
    if (!repository) return;
    if (added.includes(repository)) {
      showInfoToast(AGENT_RESOURCES_COPY.crossRepoAlreadyAdded);
      setAdding(false);
      return;
    }
    try {
      await createResource.mutateAsync({
        kind: "KIND_CONTEXT_REPO",
        enabled: true,
        repository,
        vcsProvider: choices.bitbucket ? "bitbucket" : "github",
      });
      showSuccessToast(AGENT_RESOURCES_COPY.crossRepoAdded);
      setAdding(false);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.crossRepoAddFailed));
    }
  };

  const remove = async (resourceId: string) => {
    try {
      await deleteResource.mutateAsync(resourceId);
      showSuccessToast(AGENT_RESOURCES_COPY.crossRepoRemoved);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.crossRepoRemoveFailed));
    }
  };

  return (
    <>
      <FactorySettingsCard
        title={AGENT_RESOURCES_COPY.crossRepoTitle}
        description={AGENT_RESOURCES_COPY.crossRepoHelper}
        action={
          <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
            <Button
              type="button"
              size="sm"
              disabled={!canUpdate}
              onClick={() => setAdding(true)}
              data-testid="agent-cross-repo-add"
            >
              {AGENT_RESOURCES_COPY.addRepository}
            </Button>
          </PermissionTooltip>
        }
        data-testid="agent-cross-repo"
      >
        {contextRepos.isLoading ? (
          <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.crossRepoLoadingResources}</p>
        ) : contextRepos.isError ? (
          <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.crossRepoResourcesLoadError}</p>
        ) : added.length > 0 ? (
          <ContextRepoList
            resources={contextRepos.data ?? []}
            host={choices.host}
            canUpdate={canUpdate}
            isSaving={isSaving}
            onRemove={remove}
          />
        ) : null}
      </FactorySettingsCard>
      <Dialog open={adding} onOpenChange={setAdding}>
        <DialogContent className="p-8 sm:max-w-2xl" data-testid="agent-cross-repo-picker">
          <DialogHeader>
            <DialogTitle>{AGENT_RESOURCES_COPY.addRepository}</DialogTitle>
            <DialogDescription>{AGENT_RESOURCES_COPY.crossRepoDialogHelper}</DialogDescription>
          </DialogHeader>
          <RepositoryChoice
            organizationId={organizationId}
            host={choices.host}
            bitbucket={choices.bitbucket}
            connected={Boolean(choices.integrationId)}
            loading={choices.isLoading}
            error={choices.isError}
            catalog={available}
            disabled={!canUpdate || createResource.isPending}
            onSelect={choose}
          />
        </DialogContent>
      </Dialog>
    </>
  );
}

function useGrantVcsAccess(organizationId: string, bitbucket: boolean) {
  const github = useGitHubOnboarding(organizationId, { poll: false });
  const bitbucketOnboarding = useBitbucketOnboarding(organizationId, { poll: false });
  const [granting, setGranting] = useState(false);

  const grantAccess = async () => {
    const startInstallation = bitbucket ? bitbucketOnboarding.startInstallation : github.startInstallation;
    const target = bitbucket ? "Bitbucket" : "GitHub";
    setGranting(true);
    const popup = openGitHubWindow();
    try {
      navigateGitHubWindow(popup, await startInstallation.mutateAsync());
    } catch (error) {
      popup?.close();
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.crossRepoGrantFailed(target)));
    } finally {
      setGranting(false);
    }
  };

  return { grantAccess, granting };
}

function RepositoryChoice({
  organizationId,
  host,
  bitbucket,
  connected,
  loading,
  error,
  catalog,
  disabled,
  onSelect,
}: {
  organizationId: string;
  host: VcsHostId;
  bitbucket: boolean;
  connected: boolean;
  loading: boolean;
  error: boolean;
  catalog: string[];
  disabled: boolean;
  onSelect: (name: string) => void;
}) {
  const { grantAccess, granting } = useGrantVcsAccess(organizationId, bitbucket);
  if (!connected) {
    return (
      <p className="text-[13px] text-muted-foreground">
        {bitbucket ? AGENT_RESOURCES_COPY.crossRepoConnectBitbucket : AGENT_RESOURCES_COPY.crossRepoConnectGitHub}
      </p>
    );
  }
  if (loading) {
    return <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.crossRepoLoading}</p>;
  }
  return (
    <div className="flex flex-col gap-6">
      <RepositoryPicker
        host={host}
        repos={catalog}
        selectedRepo={null}
        onSelect={onSelect}
        disabled={disabled}
        listClassName="max-h-96 min-h-72"
      />
      <FirstRunMissingAccessLine
        hint={AGENT_RESOURCES_COPY.crossRepoMissingHint}
        question={AGENT_RESOURCES_COPY.crossRepoMissingQuestion}
        action={bitbucket ? AGENT_RESOURCES_COPY.crossRepoGrantBitbucket : AGENT_RESOURCES_COPY.crossRepoGrantGitHub}
        disabled={disabled}
        loading={granting}
        loadingText={
          bitbucket ? AGENT_RESOURCES_COPY.crossRepoOpeningBitbucket : AGENT_RESOURCES_COPY.crossRepoOpeningGitHub
        }
        onClick={() => void grantAccess()}
        testId="agent-cross-repo-grant-access"
      />
      {error ? <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.crossRepoLoadError}</p> : null}
    </div>
  );
}
