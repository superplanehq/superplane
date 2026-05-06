import { ArrowDown, ArrowUp, MoreVertical, Palette, Pencil, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "../../components/Dialog/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "../../ui/dropdownMenu";
import { PermissionTooltip } from "@/components/PermissionGate";
import {
  useDeleteCanvasGroup,
  useUpdateCanvasGroup,
  useUpdateCanvasGroupPosition,
  type CanvasGroupColor,
} from "../../hooks/useCanvasData";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast, showSuccessToast } from "../../lib/toast";
import { getApiErrorMessage } from "../../lib/errors";
import { ColorSwatchPicker } from "./ColorSwatchPicker";
import type { CanvasGroupData } from "./shared";

interface CanvasGroupActionsMenuProps {
  group: CanvasGroupData;
  organizationId: string;
  canUpdateCanvases: boolean;
  permissionsLoading: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onRenameRequest: () => void;
}

export function CanvasGroupActionsMenu({
  group,
  organizationId,
  canUpdateCanvases,
  permissionsLoading,
  canMoveUp,
  canMoveDown,
  onRenameRequest,
}: CanvasGroupActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [shouldStartRename, setShouldStartRename] = useState(false);
  const [shouldOpenDeleteDialog, setShouldOpenDeleteDialog] = useState(false);
  const updateCanvasGroupMutation = useUpdateCanvasGroup(organizationId);
  const updateCanvasGroupPositionMutation = useUpdateCanvasGroupPosition(organizationId);
  const deleteCanvasGroupMutation = useDeleteCanvasGroup(organizationId);
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

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const handleColorChange = async (backgroundColor: CanvasGroupColor) => {
    if (!canUpdateCanvases || backgroundColor === group.backgroundColor) return;

    try {
      await updateCanvasGroupMutation.mutateAsync({
        groupId: group.id,
        title: group.title,
        backgroundColor,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update group color"));
    }
  };

  const handleDelete = async () => {
    if (!canUpdateCanvases) return;

    try {
      await deleteCanvasGroupMutation.mutateAsync(group.id);
      showSuccessToast("Group removed");
      closeDialog();
    } catch {
      showErrorToast("Failed to remove group");
    }
  };

  const handleRenameRequest = () => {
    setShouldStartRename(true);
    setIsMenuOpen(false);
  };

  const handleMove = async (direction: "DIRECTION_UP" | "DIRECTION_DOWN") => {
    if (!canUpdateCanvases) return;

    try {
      await updateCanvasGroupPositionMutation.mutateAsync({
        groupId: group.id,
        direction,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to move group"));
    }
  };

  const handleOpenDeleteDialog = () => {
    setShouldOpenDeleteDialog(true);
    setIsMenuOpen(false);
  };

  return (
    <>
      {!canUpdateCanvases ? (
        <PermissionTooltip allowed={allowed} message="You don't have permission to update canvases.">
          <button
            className="rounded p-1 text-white/80 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
            aria-label="Group actions"
            disabled
          >
            <MoreVertical size={16} />
          </button>
        </PermissionTooltip>
      ) : (
        <DropdownMenu open={isMenuOpen} onOpenChange={setIsMenuOpen}>
          <DropdownMenuTrigger asChild>
            <button
              className="rounded p-1 text-white/80 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
              aria-label="Group actions"
              disabled={
                updateCanvasGroupMutation.isPending ||
                updateCanvasGroupPositionMutation.isPending ||
                deleteCanvasGroupMutation.isPending
              }
            >
              <MoreVertical size={16} />
            </button>
          </DropdownMenuTrigger>
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
                void handleMove("DIRECTION_UP");
              }}
              disabled={
                !canMoveUp ||
                updateCanvasGroupMutation.isPending ||
                updateCanvasGroupPositionMutation.isPending ||
                deleteCanvasGroupMutation.isPending
              }
            >
              <ArrowUp size={16} />
              Move Up
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                void handleMove("DIRECTION_DOWN");
              }}
              disabled={
                !canMoveDown ||
                updateCanvasGroupMutation.isPending ||
                updateCanvasGroupPositionMutation.isPending ||
                deleteCanvasGroupMutation.isPending
              }
            >
              <ArrowDown size={16} />
              Move Down
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                handleRenameRequest();
              }}
              disabled={
                updateCanvasGroupMutation.isPending ||
                updateCanvasGroupPositionMutation.isPending ||
                deleteCanvasGroupMutation.isPending
              }
            >
              <Pencil size={16} />
              Change group name
            </DropdownMenuItem>
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Palette size={16} />
                Background
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="w-auto">
                <div className="p-2">
                  <ColorSwatchPicker
                    selectedColor={group.backgroundColor}
                    onSelect={(color) => void handleColorChange(color)}
                    isColorDisabled={(color) =>
                      color === group.backgroundColor ||
                      updateCanvasGroupMutation.isPending ||
                      updateCanvasGroupPositionMutation.isPending
                    }
                  />
                </div>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                handleOpenDeleteDialog();
              }}
              disabled={deleteCanvasGroupMutation.isPending}
            >
              <Trash2 size={16} />
              Remove Group
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}

      <Dialog open={isDialogOpen} onClose={closeDialog} size="lg" className="text-left">
        <DialogTitle className="text-gray-800 dark:text-white">Remove "{group.title}"?</DialogTitle>
        <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
          This will remove only the folder. The canvases will remain available.
        </DialogDescription>
        <DialogActions>
          <LoadingButton
            variant="default"
            onClick={handleDelete}
            disabled={!canUpdateCanvases}
            loading={deleteCanvasGroupMutation.isPending}
            loadingText="Removing..."
            className="flex items-center gap-2"
          >
            <Trash2 size={16} />
            Remove Group
          </LoadingButton>
          <Button variant="outline" onClick={closeDialog}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
