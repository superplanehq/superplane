import type { FactoriesFactory } from "@/api-client";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreVertical, Pencil, Trash2 } from "lucide-react";
import { useState, type MouseEvent } from "react";
import { EditFactoryDialog } from "./EditFactoryDialog";

interface FactoryDetailHeaderProps {
  factory: FactoriesFactory;
  workOrdersCount: number;
  canCreate: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  permissionsLoading: boolean;
  createHref: string;
  isUpdating: boolean;
  isDeleting: boolean;
  onUpdate: (input: { name: string; description: string }) => Promise<void>;
  onDelete: () => Promise<void>;
}

export function FactoryDetailHeader({
  factory,
  workOrdersCount: _workOrdersCount,
  canCreate,
  canUpdate,
  canDelete,
  permissionsLoading,
  createHref,
  isUpdating,
  isDeleting,
  onUpdate,
  onDelete,
}: FactoryDetailHeaderProps) {
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const canManage = canUpdate || canDelete;

  return (
    <>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <div className="flex items-start gap-2">
            <h1 className="min-w-0 text-3xl font-semibold tracking-tight text-gray-900 dark:text-gray-100">
              {factory.name}
            </h1>
            {!canManage ? (
              <PermissionTooltip
                allowed={permissionsLoading}
                message="You don't have permission to manage this factory."
              >
                <button
                  type="button"
                  className="mt-1 rounded p-1 text-gray-500 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-400"
                  aria-label="Factory actions"
                  disabled
                  data-testid="factory-actions-menu"
                >
                  <MoreVertical size={16} />
                </button>
              </PermissionTooltip>
            ) : (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    type="button"
                    className="mt-1 rounded p-1 text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
                    aria-label="Factory actions"
                    disabled={isDeleting}
                    data-testid="factory-actions-menu"
                  >
                    <MoreVertical size={16} />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  <PermissionTooltip allowed={canUpdate} message="You don't have permission to update factories.">
                    <DropdownMenuItem
                      onClick={(event: MouseEvent<HTMLElement>) => {
                        event.preventDefault();
                        if (!canUpdate) return;
                        setEditOpen(true);
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
                      onClick={(event: MouseEvent<HTMLElement>) => {
                        event.preventDefault();
                        if (!canDelete) return;
                        setDeleteOpen(true);
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
            )}
          </div>
          {factory.description?.trim() ? (
            <p className="mt-3 max-w-3xl text-sm leading-relaxed text-gray-500 dark:text-gray-400">
              {factory.description.trim()}
            </p>
          ) : null}
        </div>

        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create work orders."
        >
          <Button
            type="button"
            asChild
            disabled={!canCreate}
            className={cn(appDarkModeClasses.primaryAction)}
            data-testid="work-order-list-create-button"
          >
            <Link href={createHref}>
              <Icon name="plus" />
              New Work Order
            </Link>
          </Button>
        </PermissionTooltip>
      </div>

      <EditFactoryDialog
        open={editOpen}
        initialName={factory.name ?? ""}
        initialDescription={factory.description ?? ""}
        isSaving={isUpdating}
        onClose={() => setEditOpen(false)}
        onSave={async (input) => {
          await onUpdate(input);
          setEditOpen(false);
        }}
      />

      <Dialog open={deleteOpen} onClose={() => setDeleteOpen(false)} size="lg" className="text-left">
        <DialogTitle className="text-gray-800 dark:text-red-100">Delete "{factory.name}"?</DialogTitle>
        <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
          This cannot be undone. Are you sure you want to continue?
        </DialogDescription>
        <DialogActions>
          <LoadingButton
            variant="destructive"
            onClick={() => {
              void (async () => {
                try {
                  await onDelete();
                  setDeleteOpen(false);
                } catch {
                  // Toast handled by caller; keep dialog open for retry.
                }
              })();
            }}
            disabled={!canDelete}
            loading={isDeleting}
            loadingText="Deleting..."
            className="flex items-center gap-2"
            data-testid="factory-delete-confirm-button"
          >
            <Trash2 size={16} />
            Delete
          </LoadingButton>
          <Button variant="outline" onClick={() => setDeleteOpen(false)} disabled={isDeleting}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
