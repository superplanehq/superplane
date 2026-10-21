import { Button } from "@/components/ui/button";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { buildOrgUserDisplayMap } from "@/lib/orgUserDisplay";
import { WorkOrderAssigneesPopover } from "../WorkOrderAssigneesPopover";

export function IntakeDefaultOwnerField({
  organizationId,
  value,
  onChange,
  disabled = false,
}: {
  organizationId: string;
  value: string;
  onChange: (ownerId: string) => void;
  disabled?: boolean;
}) {
  const { data: users = [], isLoading, isError } = useOrganizationUsers(organizationId);
  const owner = buildOrgUserDisplayMap(users).get(value);
  const label = value ? (isLoading ? "Loading member…" : (owner?.name ?? "Unavailable member")) : "No owner";

  return (
    <section className="flex flex-col gap-2" aria-label="Default task owner">
      <p className="text-sm font-medium" id="intake-default-owner-label">
        Default task owner
      </p>
      <p className="text-sm text-muted-foreground" id="intake-default-owner-help">
        Assign this person when intake creates a task. Existing owner assignment rules take precedence.
      </p>
      <WorkOrderAssigneesPopover
        organizationId={organizationId}
        selectedIds={value ? [value] : []}
        align="start"
        disabled={disabled}
        onChange={(ids) => onChange(ids[0] ?? "")}
      >
        <Button
          type="button"
          variant="outline"
          className="justify-start"
          aria-labelledby="intake-default-owner-label"
          aria-describedby="intake-default-owner-help"
          disabled={disabled}
        >
          {label}
        </Button>
      </WorkOrderAssigneesPopover>
      {isError ? (
        <p role="alert">SuperPlane could not load members. Close and reopen the settings to try again.</p>
      ) : null}
      {value && !isLoading && !owner ? (
        <p className="text-sm text-muted-foreground">This member is unavailable. Select another member or No owner.</p>
      ) : null}
    </section>
  );
}
