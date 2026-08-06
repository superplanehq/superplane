import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { Settings2 } from "lucide-react";
import { OrgUserReference } from "./OrgUserReference";
import { WorkOrderAssigneesPopover } from "./WorkOrderAssigneesPopover";

interface WorkOrderAssigneesFieldProps {
  organizationId: string;
  assigneeIds: string[];
  assigneeNames: string[];
  canEdit: boolean;
  isSaving: boolean;
  onSave: (assigneeIds: string[]) => Promise<void>;
}

export function WorkOrderAssigneesField({
  organizationId,
  assigneeIds,
  assigneeNames,
  canEdit,
  isSaving,
  onSave,
}: WorkOrderAssigneesFieldProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);

  return (
    <section>
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-xs font-semibold text-gray-900 dark:text-gray-100">Assignees</h2>
        <PermissionTooltip allowed={canEdit} message="You don't have permission to update assignees.">
          <WorkOrderAssigneesPopover
            organizationId={organizationId}
            selectedIds={assigneeIds}
            onSave={onSave}
            isSaving={isSaving}
            canEdit={canEdit}
            align="end"
          >
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
              disabled={!canEdit || isSaving}
              aria-label="Edit assignees"
              data-testid="work-order-edit-assignees"
            >
              <Settings2 className="h-4 w-4" aria-hidden />
            </Button>
          </WorkOrderAssigneesPopover>
        </PermissionTooltip>
      </div>

      {assigneeIds.length > 0 ? (
        <ul className="mt-2 space-y-2">
          {assigneeIds.map((assigneeId, index) => (
            <li key={assigneeId}>
              <OrgUserReference
                display={resolveUser(assigneeId, assigneeNames[index])}
                size="md"
                nameClassName="text-sm text-gray-700 dark:text-gray-300"
              />
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">No one assigned</p>
      )}
    </section>
  );
}
