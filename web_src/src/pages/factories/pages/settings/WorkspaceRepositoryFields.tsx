import { useMemo } from "react";

import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useIntegrationResources } from "@/hooks/useIntegrations";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { BranchPicker } from "./BranchPicker";
import { branchNames, repositoryBranchChoices } from "./useFactoryRepositoryDraft";
import {
  workspaceRepositoryAccess,
  type WorkspaceRepositoryAccess,
  type WorkspaceRepositorySettings,
} from "./workspaceRepositorySettings";

type RepositoryBranchProps = {
  id: string;
  organizationId: string;
  integrationId: string;
  repository: string;
  /** Saved branch. Empty means the default branch of the repository. */
  branch: string;
  /** Default branch of the repository on the host, when known. */
  defaultBranch: string;
  disabled: boolean;
  onChange: (branch: string) => void;
};

export function WorkspaceRepositoryFields({
  idPrefix,
  value,
  showAccess,
  onChange,
  ...branch
}: Omit<RepositoryBranchProps, "id" | "branch" | "onChange"> & {
  idPrefix: string;
  value: WorkspaceRepositorySettings;
  showAccess: boolean;
  onChange: (next: WorkspaceRepositorySettings) => void;
}) {
  const accessId = `${idPrefix}-access`;
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <RepositoryBranchField
        {...branch}
        id={`${idPrefix}-branch`}
        branch={value.ref}
        onChange={(ref) => onChange({ ...value, ref })}
      />
      {showAccess ? (
        <RepositoryAccessField
          id={accessId}
          value={value.access}
          disabled={branch.disabled}
          onChange={(access) => onChange({ ...value, access })}
        />
      ) : null}
    </div>
  );
}

export function RepositoryAccessField(props: {
  id: string;
  value: WorkspaceRepositoryAccess;
  disabled: boolean;
  onChange: (access: WorkspaceRepositoryAccess) => void;
}) {
  return (
    <div className="min-w-0 space-y-2">
      <Label htmlFor={props.id}>{AGENT_RESOURCES_COPY.accessLabel}</Label>
      <Select
        value={props.value}
        disabled={props.disabled}
        onValueChange={(access) => props.onChange(workspaceRepositoryAccess(access))}
      >
        <SelectTrigger id={props.id} className="w-full" data-testid={props.id}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="write">{AGENT_RESOURCES_COPY.accessWrite}</SelectItem>
          <SelectItem value="read">{AGENT_RESOURCES_COPY.accessRead}</SelectItem>
        </SelectContent>
      </Select>
      <p className="text-[12px] text-muted-foreground">
        {props.value === "write" ? AGENT_RESOURCES_COPY.accessWriteHint : AGENT_RESOURCES_COPY.accessReadHint}
      </p>
    </div>
  );
}

export function RepositoryBranchField(props: RepositoryBranchProps) {
  return (
    <div className="min-w-0 space-y-2">
      <Label htmlFor={props.id}>{AGENT_RESOURCES_COPY.branchLabel}</Label>
      <RepositoryBranchControl {...props} />
    </div>
  );
}

function RepositoryBranchControl(props: RepositoryBranchProps) {
  const branches = useIntegrationResources(
    props.organizationId,
    props.integrationId,
    "branch",
    { repository: props.repository },
    { enabled: Boolean(props.integrationId && props.repository) },
  );
  const shown = props.branch || props.defaultBranch;
  const listed = useMemo(() => branchNames(branches.data), [branches.data]);
  const choices = useMemo(() => repositoryBranchChoices(listed, shown), [listed, shown]);

  return (
    <BranchPicker
      id={props.id}
      branch={shown}
      choices={choices}
      disabled={props.disabled}
      loading={branches.isLoading}
      allowCustomBranch
      onChange={props.onChange}
      emptyChoiceLabel={props.defaultBranch ? undefined : AGENT_RESOURCES_COPY.defaultBranch}
    />
  );
}
