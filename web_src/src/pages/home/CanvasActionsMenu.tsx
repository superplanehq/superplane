import { PermissionTooltip } from "@/components/PermissionGate";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { useCreateCanvas, useDeleteCanvas, useUpdateCanvasFolderMembership, canvasKeys } from "@/hooks/useCanvasData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { Copy, MoreVertical, Pencil, Trash2 } from "lucide-react";
import { useRef, useState, type MouseEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CanvasFolderSubmenu } from "./CanvasFolderSubmenu";
import { duplicateCanvas } from "./duplicateCanvas";
import type { CanvasCardData, CanvasFolderData } from "./types";

type PendingDuplicateCanvas = { canvasId: string; name: string };
type FolderMembershipMutation = ReturnType<typeof useUpdateCanvasFolderMembership>;

async function addDuplicateToFolder(
  canvasId: string,
  folderId: string,
  canvasFolders: CanvasFolderData[],
  mutation: FolderMembershipMutation,
) {
  const folder = canvasFolders.find((f) => f.id === folderId);
  if (!folder) return;
  try {
    await mutation.mutateAsync({
      folderId: folder.id,
      title: folder.title,
      backgroundColor: folder.backgroundColor,
      canvasIds: [...folder.canvasIds, canvasId],
    });
  } catch {
    showErrorToast("Canvas duplicated, but could not add it to the folder");
  }
}

interface CanvasActionsMenuProps {
  canvas: CanvasCardData;
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canCreateCanvases: boolean;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
  allCanvasNames: string[];
}

export function CanvasActionsMenu({
  canvas,
  canvasFolders,
  organizationId,
  onEdit,
  canCreateCanvases,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
  allCanvasNames,
}: CanvasActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const deleteCanvasMutation = useDeleteCanvas(organizationId);
  const createCanvasMutation = useCreateCanvas(organizationId);
  const updateFolderMembership = useUpdateCanvasFolderMembership(organizationId);
  const queryClient = useQueryClient();
  const pendingDuplicate = useRef(new Map<string, PendingDuplicateCanvas>());
  const sessionDuplicateNames = useRef(new Set<string>());
  const canManage = canUpdateCanvases || canDeleteCanvases || canCreateCanvases;

  const duplicateMutation = useMutation({
    mutationFn: () => {
      const pending = pendingDuplicate.current.get(canvas.id);
      return duplicateCanvas({
        sourceCanvasId: canvas.id,
        sourceName: canvas.name,
        sourceDescription: canvas.description,
        createCanvas: createCanvasMutation.mutateAsync,
        existingCanvasNames: [...allCanvasNames, ...sessionDuplicateNames.current],
        pendingCanvasId: pending?.canvasId,
        pendingCanvasName: pending?.name,
        onCanvasCreated: (canvasId, name) => {
          pendingDuplicate.current.set(canvas.id, { canvasId, name });
          sessionDuplicateNames.current.add(name);
        },
      });
    },
    onSuccess: (newCanvasId) => {
      pendingDuplicate.current.delete(canvas.id);
      queryClient.invalidateQueries({ queryKey: canvasKeys.lists() });
      queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, newCanvasId) });
      showSuccessToast("Canvas duplicated");
      if (canvas.canvasFolderId) {
        void addDuplicateToFolder(newCanvasId, canvas.canvasFolderId, canvasFolders, updateFolderMembership);
      }
    },
    onError: (error) => {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to duplicate canvas"));
    },
  });

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

  const handleDuplicate = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!canCreateCanvases) return;
    duplicateMutation.mutate();
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
                disabled={deleteCanvasMutation.isPending || duplicateMutation.isPending}
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

              <PermissionTooltip allowed={canCreateCanvases} message="You don't have permission to create canvases.">
                <DropdownMenuItem
                  onClick={handleDuplicate}
                  disabled={!canCreateCanvases || duplicateMutation.isPending}
                >
                  <Copy size={16} />
                  Duplicate
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
