import { useState } from "react";
import { ChevronDown, GitBranch, X } from "lucide-react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  useCreateFactoryAgentResource,
  useDeleteFactoryAgentResource,
  useFactoryAgentResources,
  useUpdateFactoryAgentResource,
} from "@/hooks/useFactoryAgentResources";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showInfoToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";

import { IntegrationChoiceIcon, RepositoryPicker } from "../onboarding/onboardingSteps";
import type { VcsHostId } from "../onboarding/onboardingFixtures";
import { FirstRunMissingAccessLine } from "../onboarding/first-run/FirstRunMissingAccessLine";
import { useBitbucketOnboarding } from "../onboarding/useBitbucketOnboarding";
import { useGitHubOnboarding } from "../onboarding/useGitHubOnboarding";
import { navigateGitHubWindow, openGitHubWindow } from "../onboarding/useFirstRunGitHub";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";
import { useWorkspaceRepositoryChoices } from "./useWorkspaceRepositoryChoices";
import { WorkspaceRepositoryFields } from "./WorkspaceRepositoryFields";
import {
  workspaceRepositoryAccess,
  workspaceRepositorySummary,
  type WorkspaceRepositorySettings,
} from "./workspaceRepositorySettings";

type RepositoryWorkspace = {
  onboarding?: {
    vcsIntegrationId?: string;
    vcsProvider?: string;
    appRepository?: string;
    appRepositoryId?: string;
    backlogRepositoryId?: string;
  } | null;
};

function contextRepoName(resource: { name?: string | null; repository?: string | null }): string {
  return resource.repository ?? resource.name ?? "";
}

function isWorkspaceMainRepository(name: string, appRepository: string | undefined): boolean {
  const main = appRepository?.trim().toLowerCase() ?? "";
  if (!main) return false;
  return name.trim().toLowerCase() === main;
}

type ContextRepoResource = {
  id?: string | null;
  name?: string | null;
  repository?: string | null;
  ref?: string | null;
  access?: string | null;
};

type ContextRepoListProps = {
  organizationId: string;
  integrationId: string;
  primaryRepository: string;
  resources: ContextRepoResource[];
  host: VcsHostId;
  showAccess: boolean;
  canUpdate: boolean;
  isSaving: boolean;
  onRemove: (resourceId: string) => void;
  onUpdate: (resourceId: string, changed: Partial<WorkspaceRepositorySettings>) => void;
};

function ContextRepoList(props: ContextRepoListProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  return (
    <ul
      className="divide-y divide-border overflow-hidden rounded-lg border border-border"
      data-testid="agent-cross-repo-list"
    >
      {props.primaryRepository ? (
        <li className="flex min-w-0 items-center gap-3 px-3 py-2.5" data-testid="agent-cross-repo-primary">
          <IntegrationChoiceIcon name={props.host} />
          <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{props.primaryRepository}</span>
          <span
            className="shrink-0 text-[12px] text-muted-foreground"
            title={AGENT_RESOURCES_COPY.primaryRepositoryHint}
          >
            {AGENT_RESOURCES_COPY.primaryRepository}
          </span>
        </li>
      ) : null}
      {props.resources.map((resource) => (
        <ContextRepoRow
          key={resource.id ?? contextRepoName(resource)}
          {...props}
          resource={resource}
          expanded={Boolean(resource.id) && expandedId === resource.id}
          onToggle={() => setExpandedId((current) => (current === resource.id ? null : (resource.id ?? null)))}
        />
      ))}
    </ul>
  );
}

function ContextRepoRow({
  resource,
  expanded,
  onToggle,
  ...props
}: ContextRepoListProps & { resource: ContextRepoResource; expanded: boolean; onToggle: () => void }) {
  const name = contextRepoName(resource);
  const settings = contextRepoSettings(resource);
  const panelId = `agent-cross-repo-panel-${resource.id}`;
  return (
    <li className="group min-w-0" data-testid={`agent-cross-repo-row-${resource.id}`}>
      <div className="flex min-w-0 items-center gap-1 pr-2 transition-colors hover:bg-accent/40">
        <Button
          type="button"
          variant="ghost"
          className="h-auto min-w-0 flex-1 justify-start gap-3 rounded-none px-3 py-2.5 text-left font-normal hover:bg-transparent"
          aria-expanded={expanded}
          aria-controls={panelId}
          aria-label={AGENT_RESOURCES_COPY.editRepository(name)}
          onClick={onToggle}
          data-testid={`agent-cross-repo-toggle-${resource.id}`}
        >
          <IntegrationChoiceIcon name={props.host} />
          <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{name}</span>
          <span className="shrink-0 truncate text-[12px] text-muted-foreground">
            {workspaceRepositorySummary(settings, props.showAccess)}
          </span>
          <ChevronDown
            className={cn("size-4 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-180")}
            aria-hidden
          />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8 shrink-0 text-muted-foreground transition-opacity [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:focus-visible:opacity-100 [@media(hover:hover)]:group-hover:opacity-100"
          disabled={!props.canUpdate || props.isSaving}
          aria-label={AGENT_RESOURCES_COPY.removeRepository}
          data-testid={`agent-cross-repo-remove-${resource.id}`}
          onClick={() => resource.id && props.onRemove(resource.id)}
        >
          <X className="size-4" strokeWidth={1.75} aria-hidden />
        </Button>
      </div>
      {expanded ? (
        <div id={panelId} className="border-t border-border bg-muted/30 px-3 py-3">
          <WorkspaceRepositoryFields
            idPrefix={`agent-cross-repo-${resource.id}`}
            organizationId={props.organizationId}
            integrationId={props.integrationId}
            repository={name}
            value={settings}
            showAccess={props.showAccess}
            disabled={!props.canUpdate || props.isSaving}
            onChange={(next) => resource.id && props.onUpdate(resource.id, changedRepositorySettings(settings, next))}
          />
        </div>
      ) : null}
    </li>
  );
}

// Each save sends only the changed field. A second change can start before
// the first save refetches the row, so the other field may be stale.
function changedRepositorySettings(
  current: WorkspaceRepositorySettings,
  next: WorkspaceRepositorySettings,
): Partial<WorkspaceRepositorySettings> {
  const changed: Partial<WorkspaceRepositorySettings> = {};
  if (next.ref !== current.ref) changed.ref = next.ref;
  if (next.access !== current.access) changed.access = next.access;
  return changed;
}

function contextRepoSettings(resource: ContextRepoResource): WorkspaceRepositorySettings {
  return { ref: resource.ref?.trim() ?? "", access: workspaceRepositoryAccess(resource.access) };
}

const NEW_REPOSITORY_SETTINGS: WorkspaceRepositorySettings = { ref: "", access: "write" };

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
  const [addAttempt, setAddAttempt] = useState(0);
  const contextRepos = useFactoryAgentResources(organizationId, factoryId, "KIND_CONTEXT_REPO");
  const createResource = useCreateFactoryAgentResource(organizationId, factoryId);
  const updateResource = useUpdateFactoryAgentResource(organizationId, factoryId);
  const deleteResource = useDeleteFactoryAgentResource(organizationId, factoryId);
  const choices = useWorkspaceRepositoryChoices(organizationId, factory);

  const added = (contextRepos.data ?? []).map(contextRepoName).filter(Boolean);
  const mainRepository = factory?.onboarding?.appRepository;
  const primaryRepository = mainRepository?.trim() ?? "";
  const otherRepos = (contextRepos.data ?? []).filter(
    (resource) => !isWorkspaceMainRepository(contextRepoName(resource), mainRepository),
  );
  const available = choices.repositories.filter(
    (name) => !added.includes(name) && !isWorkspaceMainRepository(name, mainRepository),
  );
  const isSaving = createResource.isPending || updateResource.isPending || deleteResource.isPending;

  const create = async (repository: string, settings: WorkspaceRepositorySettings) => {
    try {
      await createResource.mutateAsync({
        kind: "KIND_CONTEXT_REPO",
        enabled: true,
        repository,
        ref: settings.ref,
        access: choices.bitbucket ? "read" : settings.access,
        vcsProvider: choices.bitbucket ? "bitbucket" : "github",
      });
      showSuccessToast(AGENT_RESOURCES_COPY.crossRepoAdded);
      setAdding(false);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.crossRepoAddFailed));
    }
  };

  const update = async (resourceId: string, changed: Partial<WorkspaceRepositorySettings>) => {
    if (changed.ref === undefined && changed.access === undefined) return;
    try {
      await updateResource.mutateAsync({ resourceId, ...changed });
      showSuccessToast(AGENT_RESOURCES_COPY.crossRepoUpdated);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.crossRepoUpdateFailed));
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
        titleClassName="text-sm font-semibold"
        description={
          choices.bitbucket ? AGENT_RESOURCES_COPY.crossRepoHelperReadOnly : AGENT_RESOURCES_COPY.crossRepoHelper
        }
        icon={<GitBranch className="size-4" strokeWidth={1.75} aria-hidden />}
        iconClassName="border-border bg-muted/60 text-foreground"
        className="shadow-sm"
        action={
          <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
            <Button
              type="button"
              size="sm"
              disabled={!canUpdate}
              onClick={() => {
                setAddAttempt((attempt) => attempt + 1);
                setAdding(true);
              }}
              data-testid="agent-cross-repo-add"
            >
              {AGENT_RESOURCES_COPY.add}
            </Button>
          </PermissionTooltip>
        }
        data-testid="agent-cross-repo"
      >
        {contextRepos.isLoading ? (
          <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.crossRepoLoadingResources}</p>
        ) : contextRepos.isError ? (
          <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.crossRepoResourcesLoadError}</p>
        ) : primaryRepository || otherRepos.length > 0 ? (
          <ContextRepoList
            organizationId={organizationId}
            integrationId={choices.integrationId}
            primaryRepository={primaryRepository}
            resources={otherRepos}
            host={choices.host}
            showAccess={!choices.bitbucket}
            canUpdate={canUpdate}
            isSaving={isSaving}
            onRemove={remove}
            onUpdate={update}
          />
        ) : null}
      </FactorySettingsCard>
      <AddRepositoryDialog
        key={addAttempt}
        open={adding}
        onOpenChange={setAdding}
        organizationId={organizationId}
        choices={choices}
        catalog={available}
        added={added}
        mainRepository={mainRepository}
        disabled={!canUpdate || createResource.isPending}
        onAdd={create}
      />
    </>
  );
}

function AddRepositoryDialog(props: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  choices: ReturnType<typeof useWorkspaceRepositoryChoices>;
  catalog: string[];
  added: string[];
  mainRepository: string | undefined;
  disabled: boolean;
  onAdd: (repository: string, settings: WorkspaceRepositorySettings) => void;
}) {
  const [selected, setSelected] = useState<string | null>(null);
  const [settings, setSettings] = useState(NEW_REPOSITORY_SETTINGS);
  const { choices } = props;

  const choose = (name: string) => {
    const repository = name.trim();
    if (!repository || isWorkspaceMainRepository(repository, props.mainRepository)) return;
    if (props.added.includes(repository)) {
      showInfoToast(AGENT_RESOURCES_COPY.crossRepoAlreadyAdded);
      props.onOpenChange(false);
      return;
    }
    setSelected(repository);
  };

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="p-8 sm:max-w-2xl" data-testid="agent-cross-repo-picker">
        <DialogHeader>
          <DialogTitle>{selected ?? AGENT_RESOURCES_COPY.addRepository}</DialogTitle>
          <DialogDescription>
            {selected
              ? AGENT_RESOURCES_COPY.crossRepoSettingsHelper
              : choices.bitbucket
                ? AGENT_RESOURCES_COPY.crossRepoDialogHelperReadOnly
                : AGENT_RESOURCES_COPY.crossRepoDialogHelper}
          </DialogDescription>
        </DialogHeader>
        {selected ? (
          <div className="flex flex-col gap-6" data-testid="agent-cross-repo-settings">
            <WorkspaceRepositoryFields
              idPrefix="agent-cross-repo-new"
              organizationId={props.organizationId}
              integrationId={choices.integrationId}
              repository={selected}
              value={settings}
              showAccess={!choices.bitbucket}
              disabled={props.disabled}
              onChange={setSettings}
            />
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" disabled={props.disabled} onClick={() => setSelected(null)}>
                {AGENT_RESOURCES_COPY.back}
              </Button>
              <Button
                type="button"
                disabled={props.disabled}
                onClick={() => props.onAdd(selected, settings)}
                data-testid="agent-cross-repo-confirm"
              >
                {AGENT_RESOURCES_COPY.addRepository}
              </Button>
            </div>
          </div>
        ) : (
          <RepositoryChoice
            organizationId={props.organizationId}
            host={choices.host}
            bitbucket={choices.bitbucket}
            connected={Boolean(choices.integrationId)}
            loading={choices.isLoading}
            error={choices.isError}
            catalog={props.catalog}
            disabled={props.disabled}
            onSelect={choose}
          />
        )}
      </DialogContent>
    </Dialog>
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
