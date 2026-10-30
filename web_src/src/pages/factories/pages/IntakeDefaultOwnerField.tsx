import { useId, useState, type Dispatch, type SetStateAction } from "react";
import { useFactoryOwnerEligibility } from "@/hooks/useFactoryOwnerEligibility";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import type { IntakeOwnerDraft } from "./useIntakeOwnerDraft";
import { Button } from "@/components/ui/button";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { buildOrgUserDisplayMap } from "@/lib/orgUserDisplay";
import { usePermissions } from "@/contexts/usePermissions";
import { CodexConnectionSetupDialog } from "../codex/CodexConnectionSetupDialog";
import { WorkOrderAssigneesPopover } from "../WorkOrderAssigneesPopover";
import { intakeSourceSupportsDefaultOwner, type IntakeSourceSettings } from "./intakeSourceSettingsModel";

export function IntakeDefaultOwnerField({
  organizationId,
  value,
  onChange,
  disabled = false,
  factoryId,
  required = false,
  prospectiveSubscription = false,
  title = "Default task owner",
  description = "Assign this person when intake creates a task.",
}: {
  organizationId: string;
  value: string;
  onChange: (ownerId: string) => void;
  disabled?: boolean;
  factoryId?: string;
  required?: boolean;
  prospectiveSubscription?: boolean;
  title?: string;
  description?: string;
}) {
  const labelId = useId();
  const helpId = useId();
  return (
    <section className="flex flex-col gap-2" aria-label={title}>
      <p className="text-sm font-medium" id={labelId}>
        {title}
        {required ? " (required)" : ""}
      </p>
      <p className="text-sm text-muted-foreground" id={helpId}>
        {description}
      </p>
      {required && factoryId ? (
        <ConnectedOwnerSelect
          organizationId={organizationId}
          factoryId={factoryId}
          value={value}
          onChange={onChange}
          disabled={disabled}
          prospectiveSubscription={prospectiveSubscription}
          labelId={labelId}
          helpId={helpId}
        />
      ) : (
        <OptionalOwnerSelect
          organizationId={organizationId}
          value={value}
          onChange={onChange}
          disabled={disabled}
          labelId={labelId}
          helpId={helpId}
        />
      )}
    </section>
  );
}

export function IntakeSourceDefaultOwner({
  sourceId,
  organizationId,
  draft,
  disabled,
  onDraftChange,
}: {
  sourceId: string;
  organizationId?: string;
  draft: IntakeSourceSettings;
  disabled?: boolean;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
}) {
  const layout = useOptionalFactoriesLayout();
  if (
    !layout?.factory?.onboarding?.codexSubscription ||
    !intakeSourceSupportsDefaultOwner(sourceId) ||
    !organizationId
  ) {
    return null;
  }

  return (
    <IntakeDefaultOwnerField
      organizationId={organizationId}
      factoryId={layout.factoryId}
      required
      value={draft.defaultOwnerId}
      disabled={disabled}
      onChange={(defaultOwnerId) => onDraftChange((current) => ({ ...current, defaultOwnerId }))}
    />
  );
}

function ConnectedOwnerSelect({
  organizationId,
  factoryId,
  value,
  onChange,
  disabled,
  prospectiveSubscription,
  labelId,
  helpId,
}: {
  organizationId: string;
  factoryId: string;
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  prospectiveSubscription: boolean;
  labelId: string;
  helpId: string;
}) {
  const eligibility = useFactoryOwnerEligibility(organizationId, factoryId, undefined, prospectiveSubscription);
  const { data: users = [] } = useOrganizationUsers(organizationId);
  const displays = buildOrgUserDisplayMap(users);
  const { currentUserId } = usePermissions();
  const members = eligibility.data?.members ?? [];
  const hasConnectedMembers = members.some((member) => member.userId && member.eligible);
  const selected = members.find((member) => member.userId === value);
  return (
    <>
      <Select
        value={value}
        onValueChange={onChange}
        disabled={disabled || eligibility.isPending || eligibility.isError || !hasConnectedMembers}
        required
      >
        <SelectTrigger aria-labelledby={labelId} aria-describedby={helpId}>
          <SelectValue placeholder="Select a connected member" />
        </SelectTrigger>
        <SelectContent>
          {value && !members.some((member) => member.userId === value) ? (
            <SelectItem value={value} disabled>
              Unavailable member
            </SelectItem>
          ) : null}
          {members
            .filter((member) => member.userId && (member.eligible || member.userId === value))
            .map((member) => (
              <SelectItem key={member.userId} value={member.userId!} disabled={!member.eligible}>
                {displays.get(member.userId!)?.name ?? "Unavailable member"}
              </SelectItem>
            ))}
        </SelectContent>
      </Select>
      {!eligibility.isPending && !eligibility.isError && !hasConnectedMembers ? (
        <p className="text-sm text-muted-foreground">No members have ChatGPT connected.</p>
      ) : null}
      {eligibility.isError ? (
        <p role="alert" className="text-sm text-destructive">
          SuperPlane could not load connected members.{" "}
          <Button variant="link" onClick={() => void eligibility.refetch()}>
            Try again
          </Button>
        </p>
      ) : null}
      {value && selected && !selected.eligible ? (
        <p className="text-sm text-muted-foreground">This owner must reconnect Codex before tasks can run.</p>
      ) : null}
      {!eligibility.isPending &&
      !eligibility.isError &&
      !members.some((member) => member.userId === currentUserId && member.eligible) ? (
        <ConnectCodexPrompt
          organizationId={organizationId}
          factoryId={factoryId}
          disabled={disabled}
          onConnected={async () => {
            // Setup refreshes eligibility before invoking this continuation.
            if (currentUserId) onChange(currentUserId);
          }}
        />
      ) : null}
    </>
  );
}

function ConnectCodexPrompt({
  organizationId,
  factoryId,
  disabled,
  onConnected,
}: {
  organizationId: string;
  factoryId: string;
  disabled: boolean;
  onConnected: () => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
      <p>Connect Codex before you select a default task owner.</p>
      <Button type="button" size="sm" variant="outline" disabled={disabled} onClick={() => setOpen(true)}>
        Connect Codex
      </Button>
      {open ? (
        <CodexConnectionSetupDialog
          organizationId={organizationId}
          factoryId={factoryId}
          open
          onOpenChange={setOpen}
          onConnected={async () => {
            await onConnected();
            setOpen(false);
          }}
        />
      ) : null}
    </div>
  );
}

export function IntakeSetupDefaultOwner({ owner, disabled }: { owner: IntakeOwnerDraft; disabled?: boolean }) {
  if (!owner.required) return null;
  return (
    <IntakeDefaultOwnerField
      organizationId={owner.organizationId}
      factoryId={owner.factoryId}
      required
      value={owner.value}
      onChange={owner.setValue}
      disabled={disabled}
    />
  );
}

function OptionalOwnerSelect({
  organizationId,
  factoryId,
  value,
  onChange,
  disabled,
  labelId,
  helpId,
}: {
  organizationId: string;
  factoryId?: string;
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  labelId: string;
  helpId: string;
}) {
  const { data: users = [], isLoading, isError } = useOrganizationUsers(organizationId);
  const owner = buildOrgUserDisplayMap(users).get(value);
  const label = value ? (isLoading ? "Loading member…" : (owner?.name ?? "Unavailable member")) : "No owner";
  return (
    <>
      <WorkOrderAssigneesPopover
        organizationId={organizationId}
        factoryId={factoryId}
        selectedIds={value ? [value] : []}
        align="start"
        disabled={disabled}
        onChange={(ids) => onChange(ids[0] ?? "")}
      >
        <Button
          type="button"
          variant="outline"
          className="w-72 max-w-full justify-start"
          aria-labelledby={labelId}
          aria-describedby={helpId}
          disabled={disabled}
        >
          {label}
        </Button>
      </WorkOrderAssigneesPopover>
      {isError ? (
        <p className="text-sm text-destructive" role="alert">
          SuperPlane could not load members. Close and reopen the settings to try again.
        </p>
      ) : null}
      {value && !isLoading && !owner ? (
        <p className="text-sm text-muted-foreground">This member is unavailable. Select another member.</p>
      ) : null}
    </>
  );
}
