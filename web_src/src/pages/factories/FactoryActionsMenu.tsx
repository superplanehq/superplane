import { PermissionTooltip } from "@/components/PermissionGate";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreVertical, Pencil, Trash2 } from "lucide-react";

interface FactoryActionsMenuProps {
  permissions: {
    canUpdate: boolean;
    canDelete: boolean;
    loading: boolean;
  };
  isDeleting: boolean;
  onEdit: () => void;
  onDelete: () => void;
}

export function FactoryActionsMenu({ permissions, isDeleting, onEdit, onDelete }: FactoryActionsMenuProps) {
  const { canUpdate, canDelete, loading } = permissions;
  const canManage = canUpdate || canDelete;

  if (!canManage) {
    return (
      <PermissionTooltip allowed={loading} message="You don't have permission to manage this factory.">
        <button
          type="button"
          className="shrink-0 rounded p-1.5 text-gray-500 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-400"
          aria-label="Factory actions"
          disabled
          data-testid="factory-actions-menu"
        >
          <MoreVertical size={24} />
        </button>
      </PermissionTooltip>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="shrink-0 rounded p-1.5 text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
          aria-label="Factory actions"
          disabled={isDeleting}
          data-testid="factory-actions-menu"
        >
          <MoreVertical size={24} />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <PermissionTooltip allowed={canUpdate} message="You don't have permission to update factories.">
          <DropdownMenuItem
            onClick={() => {
              if (!canUpdate) return;
              onEdit();
            }}
            disabled={!canUpdate}
            data-testid="factory-edit-action"
          >
            <Pencil size={16} />
            Edit
          </DropdownMenuItem>
        </PermissionTooltip>
        <PermissionTooltip allowed={canDelete} message="You don't have permission to delete factories.">
          <DropdownMenuItem
            onClick={() => {
              if (!canDelete) return;
              onDelete();
            }}
            disabled={!canDelete}
            data-testid="factory-delete-action"
          >
            <Trash2 size={16} />
            Delete
          </DropdownMenuItem>
        </PermissionTooltip>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
