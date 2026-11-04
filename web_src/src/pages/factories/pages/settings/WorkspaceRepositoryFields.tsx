import { useMemo } from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useIntegrationResources } from "@/hooks/useIntegrations";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { BranchPicker } from "./BranchPicker";
import { branchNames, repositoryBranchChoices } from "./useFactoryRepositoryDraft";
import { workspaceRepositoryAccess, type WorkspaceRepositorySettings } from "./workspaceRepositorySettings";

export function WorkspaceRepositoryFields({
  idPrefix,
  organizationId,
  integrationId,
  repository,
  value,
  showAccess,
  disabled,
  onChange,
}: {
  idPrefix: string;
  organizationId: string;
  integrationId: string;
  repository: string;
  value: WorkspaceRepositorySettings;
  showAccess: boolean;
  disabled: boolean;
  onChange: (next: WorkspaceRepositorySettings) => void;
}) {
  const branchId = `${idPrefix}-branch`;
  const accessId = `${idPrefix}-access`;
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div className="min-w-0 space-y-2">
        <Label htmlFor={branchId}>{AGENT_RESOURCES_COPY.branchLabel}</Label>
        <RepositoryBranchControl
          id={branchId}
          organizationId={organizationId}
          integrationId={integrationId}
          repository={repository}
          branch={value.ref}
          disabled={disabled}
          onChange={(ref) => onChange({ ...value, ref })}
        />
      </div>
      {showAccess ? (
        <div className="min-w-0 space-y-2">
          <Label htmlFor={accessId}>{AGENT_RESOURCES_COPY.accessLabel}</Label>
          <Select
            value={value.access}
            disabled={disabled}
            onValueChange={(access) => onChange({ ...value, access: workspaceRepositoryAccess(access) })}
          >
            <SelectTrigger id={accessId} className="w-full" data-testid={accessId}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="write">{AGENT_RESOURCES_COPY.accessWrite}</SelectItem>
              <SelectItem value="read">{AGENT_RESOURCES_COPY.accessRead}</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-[12px] text-muted-foreground">
            {value.access === "write" ? AGENT_RESOURCES_COPY.accessWriteHint : AGENT_RESOURCES_COPY.accessReadHint}
          </p>
        </div>
      ) : null}
    </div>
  );
}

function RepositoryBranchControl(props: {
  id: string;
  organizationId: string;
  integrationId: string;
  repository: string;
  branch: string;
  disabled: boolean;
  onChange: (branch: string) => void;
}) {
  const branches = useIntegrationResources(
    props.organizationId,
    props.integrationId,
    "branch",
    { repository: props.repository },
    { enabled: Boolean(props.integrationId && props.repository) },
  );
  const listed = useMemo(() => branchNames(branches.data), [branches.data]);
  const choices = useMemo(() => repositoryBranchChoices(listed, props.branch), [listed, props.branch]);

  if (listed.length > 0) {
    return (
      <BranchPicker
        id={props.id}
        branch={props.branch}
        choices={choices}
        disabled={props.disabled}
        onChange={props.onChange}
        emptyChoiceLabel={AGENT_RESOURCES_COPY.defaultBranch}
      />
    );
  }
  return (
    <Input
      key={props.branch}
      id={props.id}
      defaultValue={props.branch}
      placeholder={AGENT_RESOURCES_COPY.defaultBranch}
      disabled={props.disabled}
      onBlur={(event) => {
        const next = event.target.value.trim();
        if (next !== props.branch) props.onChange(next);
      }}
      data-testid={props.id}
    />
  );
}
