import { PermissionTooltip } from "@/components/PermissionGate";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { useDeleteCanvas } from "@/hooks/useCanvasData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { MoreVertical, Pencil, Trash2 } from "lucide-react";
import { useState, type MouseEvent } from "react";
import type { CanvasCardData } from "./types";

interface CanvasActionsMenuProps {
  canvas: CanvasCardData;
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasDeleteDialog({
  canvasName,
  isOpen,
  isPending,
  canDelete,
  onClose,
  onDelete,
}: {
  canvasName: string;
  isOpen: boolean;
  isPending: boolean;
  canDelete: boolean;
  onClose: () => void;
  onDelete: () => void;
}) {
  return (
    <Dialog open={isOpen} onClose={onClose} size="lg" className="text-left">
      <DialogTitle className="text-gray-800 dark:text-red-100">Delete "{canvasName}"?</DialogTitle>
      <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
        This cannot be undone. Are you sure you want to continue?
      </DialogDescription>
      <DialogActions>
        <LoadingButton
          variant="destructive"
          onClick={(event) => {
            event.stopPropagation();
            onDelete();
          }}
          disabled={!canDelete}
          loading={isPending}
          loadingText="Deleting..."
          className="flex items-center gap-2"
        >
          <Trash2 size={16} />
          Delete
        </LoadingButton>
        <Button
          variant="outline"
          onClick={(event) => {
            event.stopPropagation();
            onClose();
          }}
        >
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function CanvasActionsTrigger({
  canManage,
  permissionsLoading,
  canUpdateCanvases,
  canDeleteCanvases,
  isPending,
  onRename,
  onOpenDelete,
}: {
  canManage: boolean;
  permissionsLoading: boolean;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  isPending: boolean;
  onRename: (event: MouseEvent<HTMLElement>) => void;
  onOpenDelete: (event: MouseEvent<HTMLElement>) => void;
}) {
  if (!canManage) {
    return (
      <PermissionTooltip allowed={permissionsLoading} message="You don't have permission to manage this canvas.">
        <button
          className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="Canvas actions"
          disabled
        >
          <MoreVertical size={16} />
        </button>
      </PermissionTooltip>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        asChild
        onClick={(event: MouseEvent<HTMLButtonElement>) => {
          event.preventDefault();
          event.stopPropagation();
        }}
      >
        <button
          className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="Canvas actions"
          disabled={isPending}
        >
          <MoreVertical size={16} />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <PermissionTooltip allowed={canUpdateCanvases} message="You don't have permission to update canvases.">
          <DropdownMenuItem onClick={onRename} disabled={!canUpdateCanvases}>
            <Pencil size={16} />
            Rename
          </DropdownMenuItem>
        </PermissionTooltip>

        <PermissionTooltip allowed={canDeleteCanvases} message="You don't have permission to delete canvases.">
          <DropdownMenuItem onClick={onOpenDelete} disabled={!canDeleteCanvases}>
            <Trash2 size={16} />
            Delete App
          </DropdownMenuItem>
        </PermissionTooltip>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function CanvasActionsMenu({
  canvas,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const deleteCanvasMutation = useDeleteCanvas(organizationId);
  const canManage = canUpdateCanvases || canDeleteCanvases;

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const openDialog = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDialogOpen(true);
  };

  const handleChangeName = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!canUpdateCanvases) return;
    onEdit(canvas);
  };

  const handleDelete = async () => {
    if (!canDeleteCanvases) return;

    try {
      await deleteCanvasMutation.mutateAsync(canvas.id);
      showSuccessToast("Canvas deleted successfully");
      closeDialog();
    } catch {
      showErrorToast("Failed to delete canvas");
    }
  };

  return (
    <>
      <div
        className="flex-shrink-0"
        onClick={(event: MouseEvent<HTMLDivElement>) => {
          event.preventDefault();
          event.stopPropagation();
        }}
      >
        <CanvasActionsTrigger
          canManage={canManage}
          permissionsLoading={permissionsLoading}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          isPending={deleteCanvasMutation.isPending}
          onRename={handleChangeName}
          onOpenDelete={openDialog}
        />
      </div>
      <CanvasDeleteDialog
        canvasName={canvas.name}
        isOpen={isDialogOpen}
        isPending={deleteCanvasMutation.isPending}
        canDelete={canDeleteCanvases}
        onClose={closeDialog}
        onDelete={handleDelete}
      />
    </>
  );
}
