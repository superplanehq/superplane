import { useState } from "react";

import { PermissionTooltip } from "@/components/PermissionGate";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { LoadingButton } from "@/components/ui/loading-button";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { ChevronDown } from "lucide-react";

import { IntegrationChoiceIcon } from "../onboarding/onboardingSteps";
import type { VcsHostId } from "../onboarding/onboardingFixtures";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { useRepositoryBranchesPrefetch, useRepositoryDefaultBranch } from "./useRepositoryDefaultBranch";
import { useFactoryRepository } from "./useFactoryRepository";
import { isWorkspaceMainRepository } from "./workspaceRepositorySettings";
import { RepositoryChoice } from "./WorkspaceRepositoriesSection";
import type { useWorkspaceRepositoryChoices } from "./useWorkspaceRepositoryChoices";
import { RepositoryBranchField } from "./WorkspaceRepositoryFields";

type CatalogRepositories = Parameters<typeof useRepositoryDefaultBranch>[0]["catalogRepositories"];

type PrimaryRepositoryProps = {
  organizationId: string;
  factoryId: string;
  integrationId: string;
  catalogRepositories: CatalogRepositories;
  savedRepository: string;
  savedBranch: string;
  host: VcsHostId;
  bitbucket: boolean;
  repositories: string[];
  repositoriesLoading: boolean;
  repositoriesError: boolean;
  canUpdate: boolean;
};

export function PrimaryRepositoryListItem(props: {
  organizationId: string;
  factoryId: string;
  repository: string;
  branch: string;
  choices: ReturnType<typeof useWorkspaceRepositoryChoices>;
  canUpdate: boolean;
}) {
  if (!props.repository) return null;
  return (
    <PrimaryRepositoryRow
      organizationId={props.organizationId}
      factoryId={props.factoryId}
      integrationId={props.choices.integrationId}
      catalogRepositories={props.choices.catalogRepositories}
      repository={props.repository}
      branch={props.branch}
      host={props.choices.host}
      bitbucket={props.choices.bitbucket}
      repositories={props.choices.repositories}
      repositoriesLoading={props.choices.isLoading}
      repositoriesError={props.choices.isError}
      canUpdate={props.canUpdate}
    />
  );
}

export function PrimaryRepositoryRow(
  props: Omit<PrimaryRepositoryProps, "savedRepository" | "savedBranch"> & { repository: string; branch: string },
) {
  const [expanded, setExpanded] = useState(false);
  const prefetch = useRepositoryBranchesPrefetch({
    organizationId: props.organizationId,
    integrationId: props.integrationId,
    repository: props.repository,
  });
  const panelId = "agent-primary-repository-panel";
  return (
    <li className="min-w-0" data-testid="agent-cross-repo-primary">
      <Button
        type="button"
        variant="ghost"
        className="h-auto w-full min-w-0 justify-start gap-3 rounded-none px-3 py-2.5 text-left font-normal hover:bg-accent/40"
        aria-expanded={expanded}
        aria-controls={panelId}
        aria-label={AGENT_RESOURCES_COPY.editRepository(props.repository)}
        onClick={() => setExpanded((open) => !open)}
        {...prefetch}
        data-testid="agent-cross-repo-primary-toggle"
      >
        <IntegrationChoiceIcon name={props.host} />
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{props.repository}</span>
        <span
          className="shrink-0 truncate text-[12px] text-muted-foreground"
          title={AGENT_RESOURCES_COPY.primaryRepositoryHint}
        >
          {props.branch ? `${props.branch} · ` : ""}
          {AGENT_RESOURCES_COPY.primaryRepository}
        </span>
        <ChevronDown
          className={cn("size-4 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-180")}
          aria-hidden
        />
      </Button>
      {expanded ? (
        <div id={panelId} className="border-t border-border bg-muted/30 px-3 py-3">
          <PrimaryRepositoryEditor
            key={`${props.organizationId}:${props.factoryId}:${props.repository}:${props.branch}`}
            {...props}
            savedRepository={props.repository}
            savedBranch={props.branch}
          />
        </div>
      ) : null}
    </li>
  );
}

function PrimaryRepositoryEditor(props: PrimaryRepositoryProps) {
  const draft = usePrimaryRepositoryDraft(props);
  return (
    <>
      <PrimaryRepositoryFields {...props} draft={draft} />
      <ChangePrimaryRepositoryDialog {...props} draft={draft} />
      <UpdatePrimaryTasksDialog
        open={draft.confirmOpen}
        description={primaryTaskMoveMessage({
          savedRepository: props.savedRepository,
          repository: draft.repository,
          savedBranch: props.savedBranch,
          branch: draft.branchToSave,
        })}
        onOpenChange={draft.setConfirmOpen}
        onUpdate={() => void draft.save(true)}
        onKeep={() => {
          draft.setConfirmOpen(false);
          void draft.save(false);
        }}
      />
    </>
  );
}

function usePrimaryRepositoryDraft(props: PrimaryRepositoryProps) {
  const [repository, setRepository] = useState(props.savedRepository);
  const [branch, setBranch] = useState(props.savedBranch);
  const [picking, setPicking] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const updateRepository = useFactoryRepository(props.organizationId, props.factoryId);
  const hostDefault = useRepositoryDefaultBranch({
    organizationId: props.organizationId,
    integrationId: props.integrationId,
    repository,
    catalogRepositories: props.catalogRepositories,
  });
  const branchToSave = branch.trim() || hostDefault;
  const dirty =
    !isWorkspaceMainRepository(repository, props.savedRepository) || branch.trim() !== props.savedBranch.trim();
  const disabled = !props.canUpdate || updateRepository.isPending;

  const save = async (retargetTasks: boolean) => {
    if (!repository.trim() || !branchToSave || disabled) return;
    try {
      await updateRepository.mutateAsync({
        repository: repository.trim(),
        defaultBranch: branchToSave,
        retargetTasks,
      });
      showSuccessToast(AGENT_RESOURCES_COPY.primaryRepositoryUpdated);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.primaryRepositoryUpdateFailed));
    }
  };

  const chooseRepository = (name: string) => {
    const next = name.trim();
    if (!next) return;
    if (!isWorkspaceMainRepository(next, repository)) {
      setRepository(next);
      setBranch("");
    }
    setPicking(false);
  };

  return {
    repository,
    branch,
    setBranch,
    picking,
    setPicking,
    confirmOpen,
    setConfirmOpen,
    branchToSave,
    dirty,
    disabled,
    saving: updateRepository.isPending,
    hostDefault,
    save,
    chooseRepository,
  };
}

type PrimaryDraft = ReturnType<typeof usePrimaryRepositoryDraft>;

function PrimaryRepositoryFields(props: PrimaryRepositoryProps & { draft: PrimaryDraft }) {
  const { draft } = props;
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <p className="min-w-0 truncate text-[13px] font-medium">{draft.repository}</p>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={draft.disabled || !props.integrationId}
          onClick={() => draft.setPicking(true)}
          data-testid="agent-primary-change-repository"
        >
          {AGENT_RESOURCES_COPY.changeRepository}
        </Button>
      </div>
      {props.integrationId ? (
        <RepositoryBranchField
          id="agent-primary-branch"
          label={AGENT_RESOURCES_COPY.defaultBranch}
          organizationId={props.organizationId}
          integrationId={props.integrationId}
          repository={draft.repository}
          branch={draft.branch}
          defaultBranch={draft.hostDefault}
          disabled={draft.disabled}
          onChange={draft.setBranch}
        />
      ) : (
        <p className="text-[13px] text-muted-foreground">
          {props.bitbucket
            ? AGENT_RESOURCES_COPY.crossRepoConnectBitbucket
            : AGENT_RESOURCES_COPY.crossRepoConnectGitHub}
        </p>
      )}
      <div className="flex justify-end">
        <PermissionTooltip allowed={props.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
          <LoadingButton
            type="button"
            loading={draft.saving}
            loadingText={AGENT_RESOURCES_COPY.savingRepository}
            disabled={draft.disabled || !draft.dirty || !draft.branchToSave}
            onClick={() => draft.setConfirmOpen(true)}
            data-testid="agent-primary-save"
          >
            {AGENT_RESOURCES_COPY.saveRepository}
          </LoadingButton>
        </PermissionTooltip>
      </div>
    </div>
  );
}

function ChangePrimaryRepositoryDialog(props: PrimaryRepositoryProps & { draft: PrimaryDraft }) {
  return (
    <Dialog open={props.draft.picking} onOpenChange={props.draft.setPicking}>
      <DialogContent className="p-8 sm:max-w-2xl" data-testid="agent-primary-repository-picker">
        <DialogHeader>
          <DialogTitle>{AGENT_RESOURCES_COPY.changeRepository}</DialogTitle>
          <DialogDescription>{AGENT_RESOURCES_COPY.primaryRepositoryHint}</DialogDescription>
        </DialogHeader>
        <RepositoryChoice
          organizationId={props.organizationId}
          host={props.host}
          bitbucket={props.bitbucket}
          connected={Boolean(props.integrationId)}
          loading={props.repositoriesLoading}
          error={props.repositoriesError}
          catalog={props.repositories}
          disabled={props.draft.disabled}
          onSelect={props.draft.chooseRepository}
        />
      </DialogContent>
    </Dialog>
  );
}

function UpdatePrimaryTasksDialog(props: {
  open: boolean;
  description: string;
  onOpenChange: (open: boolean) => void;
  onUpdate: () => void;
  onKeep: () => void;
}) {
  return (
    <AlertDialog open={props.open} onOpenChange={props.onOpenChange}>
      <AlertDialogContent data-testid="agent-primary-update-tasks-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>{AGENT_RESOURCES_COPY.updateUnfinishedTasks}</AlertDialogTitle>
          <AlertDialogDescription>{props.description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{AGENT_RESOURCES_COPY.cancel}</AlertDialogCancel>
          <Button type="button" variant="outline" data-testid="agent-primary-keep-tasks" onClick={props.onKeep}>
            {AGENT_RESOURCES_COPY.keepTasks}
          </Button>
          <AlertDialogAction onClick={props.onUpdate} data-testid="agent-primary-update-tasks">
            {AGENT_RESOURCES_COPY.updateTasks}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function primaryTaskMoveMessage(args: {
  savedRepository: string;
  repository: string;
  savedBranch: string;
  branch: string;
}): string {
  const savedRepository = args.savedRepository.trim();
  const repository = args.repository.trim();
  const savedBranch = args.savedBranch.trim();
  const branch = args.branch.trim();
  const move = isWorkspaceMainRepository(repository, savedRepository)
    ? branchTaskMove(savedBranch, branch)
    : repositoryTaskMove(savedRepository, repository, savedBranch, branch);
  return `${move} ${AGENT_RESOURCES_COPY.keepTasksResult}`;
}

function repositoryTaskMove(
  previousRepository: string,
  nextRepository: string,
  previousBranch: string,
  nextBranch: string,
) {
  if (previousBranch && previousBranch !== nextBranch) {
    return `Unfinished tasks on "${previousRepository}" with no branch, or with "${previousBranch}", will use "${nextRepository}" and "${nextBranch}". Closed tasks stay unchanged.`;
  }
  if (previousBranch) {
    return `Unfinished tasks on "${previousRepository}" will use "${nextRepository}". Closed tasks stay unchanged.`;
  }
  return `Unfinished tasks on "${previousRepository}" with no branch will use "${nextRepository}" and "${nextBranch}". Closed tasks stay unchanged.`;
}

function branchTaskMove(previousBranch: string, nextBranch: string) {
  if (previousBranch) {
    return `Unfinished tasks with no branch, or with "${previousBranch}", will use "${nextBranch}". Closed tasks stay unchanged.`;
  }
  return `Unfinished tasks with no branch will use "${nextBranch}". Closed tasks stay unchanged.`;
}
