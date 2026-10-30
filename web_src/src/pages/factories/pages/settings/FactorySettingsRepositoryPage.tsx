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
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Check, ChevronDown } from "lucide-react";
import { useState, type ReactNode } from "react";
import { RepositoryPicker } from "../onboarding/onboardingSteps";
import type { VcsHostId } from "../onboarding/onboardingFixtures";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { useFactoryRepositoryDraft } from "./useFactoryRepositoryDraft";
import { useWorkspaceRepositoryChoices } from "./useWorkspaceRepositoryChoices";

const BRANCH_FIELD_ID = "factory-settings-default-branch";

export function FactorySettingsRepositoryPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const choices = useWorkspaceRepositoryChoices(organizationId, factory);
  const draft = useFactoryRepositoryDraft({
    organizationId,
    factoryId,
    factory,
    integrationId: choices.integrationId,
    catalogRepositories: choices.catalogRepositories,
  });
  const canUpdate = canAct("factories", "update");
  const [confirmTasks, setConfirmTasks] = useState(false);

  usePageTitle(["Repository", "Settings", factory.name ?? "Workspace"]);
  const requestSave = () => {
    const repositoryChanged = !sameRepositoryName(draft.repository, draft.savedRepository);
    const branchChanged = draft.branch.trim() !== draft.savedBranch;
    if (repositoryChanged || branchChanged) {
      setConfirmTasks(true);
      return;
    }
    void draft.save(true);
  };

  return (
    <FactorySettingsPageFrame title="Repository" subtitle={repositorySettingsSubtitle(choices.bitbucket)}>
      <FactorySettingsCard title={repositorySettingsTitle(choices.bitbucket)} data-testid="factory-settings-repository">
        {repositorySettingsBody(choices, draft.repository, draft.selectRepository)}
        {choices.integrationId ? (
          <DefaultBranchField
            branch={draft.branch}
            choices={draft.branchChoices}
            listedBranchNames={draft.listedBranchNames}
            disabled={!canUpdate || draft.isSaving || !choices.integrationId}
            onChange={draft.setBranch}
          />
        ) : null}
        {choices.isError ? (
          <p className="mt-3 text-[13px] text-destructive">We could not load repositories. Try again.</p>
        ) : null}
        <div className="mt-4 flex items-center justify-between gap-4 border-t border-border pt-4">
          <p className="text-[12px] text-muted-foreground">{repositorySettingsFooter(choices.bitbucket)}</p>
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You do not have permission to update this workspace."
          >
            <LoadingButton
              type="button"
              loading={draft.isSaving}
              loadingText="Saving..."
              disabled={
                !draft.repository ||
                !draft.branch ||
                !draft.isDirty ||
                !canUpdate ||
                permissionsLoading ||
                draft.isSaving
              }
              onClick={requestSave}
              data-testid="factory-settings-repository-save"
            >
              Save repository
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </FactorySettingsCard>
      <UpdateUnfinishedTasksDialog
        open={confirmTasks}
        previousRepository={draft.savedRepository}
        nextRepository={draft.repository}
        previousBranch={draft.savedBranch}
        nextBranch={draft.branch.trim()}
        onOpenChange={setConfirmTasks}
        onConfirm={() => {
          setConfirmTasks(false);
          void draft.save(true);
        }}
        onKeepTasks={() => {
          setConfirmTasks(false);
          void draft.save(false);
        }}
      />
    </FactorySettingsPageFrame>
  );
}

function UpdateUnfinishedTasksDialog(props: {
  open: boolean;
  previousRepository: string;
  nextRepository: string;
  previousBranch: string;
  nextBranch: string;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  onKeepTasks: () => void;
}) {
  return (
    <AlertDialog open={props.open} onOpenChange={props.onOpenChange}>
      <AlertDialogContent data-testid="factory-settings-update-tasks-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>Update unfinished tasks</AlertDialogTitle>
          <AlertDialogDescription>{unfinishedTasksDescription(props)}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel data-testid="factory-settings-update-tasks-cancel">Cancel</AlertDialogCancel>
          <Button
            type="button"
            variant="outline"
            data-testid="factory-settings-update-tasks-keep"
            onClick={props.onKeepTasks}
          >
            Save without updating tasks
          </Button>
          <AlertDialogAction onClick={props.onConfirm} data-testid="factory-settings-update-tasks-confirm">
            Update tasks
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function unfinishedTasksDescription(props: {
  previousRepository: string;
  nextRepository: string;
  previousBranch: string;
  nextBranch: string;
}) {
  const previousRepository = props.previousRepository.trim();
  const nextRepository = props.nextRepository.trim();
  const previousBranch = props.previousBranch.trim();
  const repositoryChanged = previousRepository !== "" && !sameRepositoryName(previousRepository, nextRepository);
  const move = repositoryChanged
    ? repositoryTaskMove(previousRepository, nextRepository, previousBranch, props.nextBranch)
    : branchTaskMove(previousBranch, props.nextBranch);
  return `${move} Save without updating tasks keeps current tasks on their repository and branch.`;
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

function sameRepositoryName(left: string, right: string) {
  return left.trim().toLowerCase() === right.trim().toLowerCase();
}

function repositorySettingsSubtitle(bitbucket: boolean) {
  if (bitbucket) return "Select the Bitbucket repository for this workspace.";
  return "Select the GitHub repository for workspace work and issue intake.";
}

function repositorySettingsTitle(bitbucket: boolean) {
  if (bitbucket) return "Bitbucket repository";
  return "GitHub repository";
}

function repositorySettingsFooter(bitbucket: boolean) {
  if (bitbucket) return "Saving updates the repository this workspace uses.";
  return "Saving updates factory GitHub issue intake and pull request automations.";
}

function repositorySettingsBody(
  choices: {
    integrationId: string;
    bitbucket: boolean;
    host: VcsHostId;
    repositories: string[];
    isLoading: boolean;
  },
  repository: string,
  onSelect: (repository: string) => void,
): ReactNode {
  if (!choices.integrationId) {
    return (
      <p className="text-[13px] text-muted-foreground">
        {choices.bitbucket
          ? "Connect Bitbucket during workspace setup before you select a repository."
          : "Connect GitHub during workspace setup before you select a repository."}
      </p>
    );
  }
  if (choices.isLoading) {
    return <p className="text-[13px] text-muted-foreground">Loading repositories...</p>;
  }
  return (
    <RepositoryPicker
      host={choices.host}
      repos={choices.repositories}
      selectedRepo={repository || null}
      onSelect={onSelect}
    />
  );
}

function DefaultBranchField(props: {
  branch: string;
  choices: string[];
  listedBranchNames: string[];
  disabled: boolean;
  onChange: (branch: string) => void;
}) {
  return (
    <div className="mt-4 space-y-2">
      <Label htmlFor={BRANCH_FIELD_ID}>Default branch</Label>
      <DefaultBranchControl {...props} />
    </div>
  );
}

function DefaultBranchControl(props: {
  branch: string;
  choices: string[];
  listedBranchNames: string[];
  disabled: boolean;
  onChange: (branch: string) => void;
}) {
  if (props.listedBranchNames.length > 0) {
    return <BranchPicker {...props} />;
  }
  return (
    <Input
      id={BRANCH_FIELD_ID}
      value={props.branch}
      disabled={props.disabled}
      onChange={(event) => props.onChange(event.target.value)}
      data-testid={BRANCH_FIELD_ID}
    />
  );
}

function BranchPicker(props: {
  branch: string;
  choices: string[];
  disabled: boolean;
  onChange: (branch: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const selectBranch = (name: string) => {
    props.onChange(name);
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          id={BRANCH_FIELD_ID}
          role="combobox"
          aria-expanded={open}
          aria-haspopup="listbox"
          disabled={props.disabled}
          data-testid={BRANCH_FIELD_ID}
          className="w-full justify-between rounded-md px-2.5 font-normal"
        >
          <span className={cn("min-w-0 truncate", props.branch ? "text-foreground" : "text-muted-foreground")}>
            {props.branch || "Select a branch"}
          </span>
          <ChevronDown className="size-4 shrink-0 opacity-50" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[var(--radix-popover-trigger-width)] p-0">
        <Command>
          <CommandInput placeholder="Find a branch" autoFocus />
          <CommandList>
            <CommandEmpty>No branch found.</CommandEmpty>
            <CommandGroup>
              {props.choices.map((name) => (
                <CommandItem key={name} value={name} onSelect={() => selectBranch(name)}>
                  <span className="min-w-0 flex-1 truncate">{name}</span>
                  <Check className={cn("size-4", name === props.branch ? "opacity-100" : "opacity-0")} aria-hidden />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
