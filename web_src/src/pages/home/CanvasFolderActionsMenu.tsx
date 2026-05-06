import { PermissionTooltip } from "@/components/PermissionGate";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import {
  CANVAS_FOLDER_COLORS,
  useDeleteCanvasFolder,
  useMoveCanvasFolder,
  useUpdateCanvasFolder,
  type CanvasFolderColor,
} from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { ArrowDown, ArrowUp, Check, MoreVertical, Palette, Pencil, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { FOLDER_COLOR_OPTIONS } from "./canvasFolderStyles";
import type { CanvasFolderData } from "./types";

interface CanvasFolderActionsMenuProps {
  folder: CanvasFolderData;
  organizationId: string;
  canUpdateCanvases: boolean;
  permissionsLoading: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onRenameRequest: () => void;
}

interface CanvasFolderMenuContentProps {
  folder: CanvasFolderData;
  canMoveUp: boolean;
  canMoveDown: boolean;
  isUpdatingFolder: boolean;
  isMovingFolder: boolean;
  isDeletingFolder: boolean;
  shouldStartRename: boolean;
  onMove: (direction: "DIRECTION_UP" | "DIRECTION_DOWN") => void;
  onRenameRequest: () => void;
  onColorChange: (backgroundColor: CanvasFolderColor) => void;
  onOpenDeleteDialog: () => void;
}

function CanvasFolderMenuContent({
  folder,
  canMoveUp,
  canMoveDown,
  isUpdatingFolder,
  isMovingFolder,
  isDeletingFolder,
  shouldStartRename,
  onMove,
  onRenameRequest,
  onColorChange,
  onOpenDeleteDialog,
}: CanvasFolderMenuContentProps) {
  const isMutating = isUpdatingFolder || isMovingFolder || isDeletingFolder;

  return (
    <DropdownMenuContent
      align="end"
      onCloseAutoFocus={(event) => {
        if (shouldStartRename) {
          event.preventDefault();
        }
      }}
    >
      <DropdownMenuItem
        onSelect={(event) => {
          event.preventDefault();
          onMove("DIRECTION_UP");
        }}
        disabled={!canMoveUp || isMutating}
      >
        <ArrowUp size={16} />
        Move Up
      </DropdownMenuItem>
      <DropdownMenuItem
        onSelect={(event) => {
          event.preventDefault();
          onMove("DIRECTION_DOWN");
        }}
        disabled={!canMoveDown || isMutating}
      >
        <ArrowDown size={16} />
        Move Down
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        onSelect={(event) => {
          event.preventDefault();
          onRenameRequest();
        }}
        disabled={isMutating}
      >
        <Pencil size={16} />
        Change folder name
      </DropdownMenuItem>
      <DropdownMenuSub>
        <DropdownMenuSubTrigger>
          <Palette size={16} />
          Background
        </DropdownMenuSubTrigger>
        <DropdownMenuSubContent className="w-auto">
          <div className="flex items-center gap-2 p-2">
            {CANVAS_FOLDER_COLORS.map((color) => (
              <button
                key={color}
                type="button"
                aria-label={`${FOLDER_COLOR_OPTIONS[color].label} folder color`}
                className={cn(
                  "flex h-6 w-6 items-center justify-center rounded-full border border-slate-950/15 text-white",
                  FOLDER_COLOR_OPTIONS[color].swatchClass,
                  folder.backgroundColor === color && "ring-2 ring-gray-900 ring-offset-1",
                )}
                onClick={() => onColorChange(color)}
                disabled={color === folder.backgroundColor || isUpdatingFolder || isMovingFolder}
              >
                {folder.backgroundColor === color ? <Check className="h-3 w-3" /> : null}
              </button>
            ))}
          </div>
        </DropdownMenuSubContent>
      </DropdownMenuSub>
      <DropdownMenuItem
        onSelect={(event) => {
          event.preventDefault();
          onOpenDeleteDialog();
        }}
        disabled={isDeletingFolder}
      >
        <Trash2 size={16} />
        Remove Folder
      </DropdownMenuItem>
    </DropdownMenuContent>
  );
}

function CanvasFolderDeleteDialog({
  folder,
  open,
  canUpdateCanvases,
  isDeleting,
  onClose,
  onDelete,
}: {
  folder: CanvasFolderData;
  open: boolean;
  canUpdateCanvases: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onDelete: () => void;
}) {
  return (
    <Dialog open={open} onClose={onClose} size="lg" className="text-left">
      <DialogTitle className="text-gray-800 dark:text-white">Remove "{folder.title}"?</DialogTitle>
      <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
        This will remove only the folder. The canvases will remain available.
      </DialogDescription>
      <DialogActions>
        <LoadingButton
          variant="default"
          onClick={onDelete}
          disabled={!canUpdateCanvases}
          loading={isDeleting}
          loadingText="Removing..."
          className="flex items-center gap-2"
        >
          <Trash2 size={16} />
          Remove Folder
        </LoadingButton>
        <Button variant="outline" onClick={onClose}>
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function DisabledCanvasFolderActionsButton({ allowed }: { allowed: boolean }) {
  return (
    <PermissionTooltip allowed={allowed} message="You don't have permission to update canvases.">
      <button
        className="rounded p-1 text-white/80 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
        aria-label="Folder actions"
        disabled
      >
        <MoreVertical size={16} />
      </button>
    </PermissionTooltip>
  );
}

export function CanvasFolderActionsMenu({
  folder,
  organizationId,
  canUpdateCanvases,
  permissionsLoading,
  canMoveUp,
  canMoveDown,
  onRenameRequest,
}: CanvasFolderActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [shouldStartRename, setShouldStartRename] = useState(false);
  const [shouldOpenDeleteDialog, setShouldOpenDeleteDialog] = useState(false);
  const updateCanvasFolderMutation = useUpdateCanvasFolder(organizationId);
  const moveCanvasFolderMutation = useMoveCanvasFolder(organizationId);
  const deleteCanvasFolderMutation = useDeleteCanvasFolder(organizationId);
  const allowed = canUpdateCanvases || permissionsLoading;

  useEffect(() => {
    if (!isMenuOpen && shouldStartRename) {
      setShouldStartRename(false);
      onRenameRequest();
    }
  }, [isMenuOpen, onRenameRequest, shouldStartRename]);

  useEffect(() => {
    if (!isMenuOpen && shouldOpenDeleteDialog) {
      setShouldOpenDeleteDialog(false);
      setIsDialogOpen(true);
    }
  }, [isMenuOpen, shouldOpenDeleteDialog]);

  const handleColorChange = async (backgroundColor: CanvasFolderColor) => {
    if (!canUpdateCanvases || backgroundColor === folder.backgroundColor) return;

    try {
      await updateCanvasFolderMutation.mutateAsync({
        folderId: folder.id,
        title: folder.title,
        backgroundColor,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update folder color"));
    }
  };

  const handleDelete = async () => {
    if (!canUpdateCanvases) return;

    try {
      await deleteCanvasFolderMutation.mutateAsync(folder.id);
      showSuccessToast("Folder removed");
      setIsDialogOpen(false);
    } catch {
      showErrorToast("Failed to remove folder");
    }
  };

  const handleRenameRequest = () => {
    setShouldStartRename(true);
    setIsMenuOpen(false);
  };

  const handleMove = async (direction: "DIRECTION_UP" | "DIRECTION_DOWN") => {
    if (!canUpdateCanvases) return;

    try {
      await moveCanvasFolderMutation.mutateAsync({
        folderId: folder.id,
        direction,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to move folder"));
    }
  };

  return (
    <>
      {!canUpdateCanvases ? (
        <DisabledCanvasFolderActionsButton allowed={allowed} />
      ) : (
        <DropdownMenu open={isMenuOpen} onOpenChange={setIsMenuOpen}>
          <DropdownMenuTrigger asChild>
            <button
              className="rounded p-1 text-white/80 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
              aria-label="Folder actions"
              disabled={
                updateCanvasFolderMutation.isPending ||
                moveCanvasFolderMutation.isPending ||
                deleteCanvasFolderMutation.isPending
              }
            >
              <MoreVertical size={16} />
            </button>
          </DropdownMenuTrigger>
          <CanvasFolderMenuContent
            folder={folder}
            canMoveUp={canMoveUp}
            canMoveDown={canMoveDown}
            isUpdatingFolder={updateCanvasFolderMutation.isPending}
            isMovingFolder={moveCanvasFolderMutation.isPending}
            isDeletingFolder={deleteCanvasFolderMutation.isPending}
            shouldStartRename={shouldStartRename}
            onMove={(direction) => void handleMove(direction)}
            onRenameRequest={handleRenameRequest}
            onColorChange={(color) => void handleColorChange(color)}
            onOpenDeleteDialog={() => {
              setShouldOpenDeleteDialog(true);
              setIsMenuOpen(false);
            }}
          />
        </DropdownMenu>
      )}

      <CanvasFolderDeleteDialog
        folder={folder}
        open={isDialogOpen}
        canUpdateCanvases={canUpdateCanvases}
        isDeleting={deleteCanvasFolderMutation.isPending}
        onClose={() => setIsDialogOpen(false)}
        onDelete={() => void handleDelete()}
      />
    </>
  );
}
