import { PermissionTooltip } from "@/components/PermissionGate";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { RepositoryPicker } from "../onboarding/onboardingSteps";
import type { VcsHostId } from "../onboarding/onboardingFixtures";
import type { ReactNode } from "react";

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

  usePageTitle(["Repository", "Settings", factory.name ?? "Workspace"]);

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
              onClick={draft.save}
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
      <p className="text-[12px] text-muted-foreground">New tasks use this branch. Started tasks keep their branch.</p>
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
    return (
      <Select value={props.branch || undefined} onValueChange={props.onChange} disabled={props.disabled}>
        <SelectTrigger id={BRANCH_FIELD_ID} className="w-full" data-testid={BRANCH_FIELD_ID}>
          <SelectValue placeholder="Select a branch" />
        </SelectTrigger>
        <SelectContent>
          {props.choices.map((name) => (
            <SelectItem key={name} value={name}>
              {name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
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
