import { PermissionTooltip } from "@/components/PermissionGate";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { useDeleteCanvas } from "@/hooks/useCanvasData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { MoreVertical, Pencil, Trash2 } from "lucide-react";
import { useState, type MouseEvent } from "react";
import { CanvasFolderSubmenu } from "./CanvasFolderSubmenu";
import type { CanvasCardData, CanvasFolderData } from "./types";

interface CanvasActionsMenuProps {
  canvas: CanvasCardData;
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

export function CanvasActionsMenu({
  canvas,
  canvasFolders,
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
        {!canManage ? (
          <PermissionTooltip allowed={permissionsLoading} message="You don't have permission to manage this canvas.">
            <button
              className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
              aria-label="Canvas actions"
              disabled
            >
              <MoreVertical size={16} />
            </button>
          </PermissionTooltip>
        ) : (
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
                disabled={deleteCanvasMutation.isPending}
              >
                <MoreVertical size={16} />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <PermissionTooltip allowed={canUpdateCanvases} message="You don't have permission to update canvases.">
                <DropdownMenuItem onClick={handleChangeName} disabled={!canUpdateCanvases}>
                  <Pencil size={16} />
                  Rename
                </DropdownMenuItem>
              </PermissionTooltip>

              <CanvasFolderSubmenu
                canvas={canvas}
                canvasFolders={canvasFolders}
                organizationId={organizationId}
                canUpdateCanvases={canUpdateCanvases}
              />

              <PermissionTooltip allowed={canDeleteCanvases} message="You don't have permission to delete canvases.">
                <DropdownMenuItem onClick={openDialog} disabled={!canDeleteCanvases}>
                  <Trash2 size={16} />
                  Delete App
                </DropdownMenuItem>
              </PermissionTooltip>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <Dialog open={isDialogOpen} onClose={closeDialog} size="lg" className="text-left">
        <DialogTitle className="text-gray-800 dark:text-red-100">Delete "{canvas.name}"?</DialogTitle>
        <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
          This cannot be undone. Are you sure you want to continue?
        </DialogDescription>
        <DialogActions>
          <LoadingButton
            variant="destructive"
            onClick={(event) => {
              event.stopPropagation();
              handleDelete();
            }}
            disabled={!canDeleteCanvases}
            loading={deleteCanvasMutation.isPending}
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
              closeDialog();
            }}
          >
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
