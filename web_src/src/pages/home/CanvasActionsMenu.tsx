import { Check, FolderMinus, FolderPlus, MoreVertical, Pencil, Trash2 } from "lucide-react";
import { useState, type FormEvent, type MouseEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router-dom";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "../../components/Dialog/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "../../ui/dropdownMenu";
import { Input } from "../../components/Input/input";
import { PermissionTooltip } from "@/components/PermissionGate";
import {
  canvasKeys,
  useCreateCanvasGroup,
  useDeleteCanvas,
  useUpdateCanvasGroupMembership,
  type CanvasGroupColor,
} from "../../hooks/useCanvasData";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "../../lib/utils";
import { showErrorToast, showSuccessToast } from "../../lib/toast";
import { getApiErrorMessage } from "../../lib/errors";
import type { CanvasesCanvasGroup } from "@/api-client";
import { ColorSwatchPicker } from "./ColorSwatchPicker";
import { GROUP_SWATCH_CLASSES, type CanvasCardData, type CanvasGroupData } from "./shared";

interface CanvasActionsMenuProps {
  canvas: CanvasCardData;
  canvasGroups: CanvasGroupData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

export function CanvasActionsMenu({
  canvas,
  canvasGroups,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [newGroupTitle, setNewGroupTitle] = useState("");
  const [newGroupColor, setNewGroupColor] = useState<CanvasGroupColor>("blue-800");
  const deleteCanvasMutation = useDeleteCanvas(organizationId);
  const createCanvasGroupMutation = useCreateCanvasGroup(organizationId);
  const updateCanvasGroupMembershipMutation = useUpdateCanvasGroupMembership(organizationId);
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const canManage = canUpdateCanvases || canDeleteCanvases;

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const openDialog = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDialogOpen(true);
  };

  const handleAssignToGroup = async (groupId: string) => {
    if (!canUpdateCanvases || groupId === canvas.canvasGroupId) return;

    try {
      await updateCanvasGroupMembershipMutation.mutateAsync({ canvasId: canvas.id, groupId });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to add canvas to group"));
    }
  };

  const handleRemoveFromGroup = async () => {
    if (!canUpdateCanvases || !canvas.canvasGroupId) return;

    try {
      await updateCanvasGroupMembershipMutation.mutateAsync({ canvasId: canvas.id });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to remove canvas from group"));
    }
  };

  const handleCreateGroup = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!canUpdateCanvases) return;

    const title = newGroupTitle.trim();
    if (!title) return;

    try {
      const response = await createCanvasGroupMutation.mutateAsync({ title, backgroundColor: newGroupColor });
      let groupId = response.data?.group?.metadata?.id;

      if (!groupId) {
        await queryClient.invalidateQueries({ queryKey: canvasKeys.groupList(organizationId) });
        await queryClient.refetchQueries({ queryKey: canvasKeys.groupList(organizationId), type: "active" });

        const groups = queryClient.getQueryData<CanvasesCanvasGroup[]>(canvasKeys.groupList(organizationId)) || [];
        groupId =
          groups.find((group) => group.spec?.title?.trim().toLowerCase() === title.toLowerCase())?.metadata?.id || "";
      }

      if (!groupId) {
        throw new Error("missing canvas group id");
      }

      await updateCanvasGroupMembershipMutation.mutateAsync({ canvasId: canvas.id, groupId });

      setNewGroupTitle("");
      setNewGroupColor("blue-800");
      showSuccessToast("Group created");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create group"));
    }
  };

  const handleDelete = async () => {
    if (!canDeleteCanvases) return;
    const currentPath = location.pathname;
    const canvasPath = `/${organizationId}/canvases/${canvas.id}`;
    const isViewingCanvas = currentPath === canvasPath || currentPath.startsWith(`${canvasPath}/`);

    if (isViewingCanvas) {
      queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvas.id) });
      navigate(`/${organizationId}`, { replace: true });
      deleteCanvasMutation.mutate(canvas.id, {
        onSuccess: () => {
          showSuccessToast("Canvas deleted successfully");
          closeDialog();
        },
        onError: () => {
          showErrorToast("Failed to delete canvas");
        },
      });
      return;
    }

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
          <PermissionTooltip
            allowed={canManage || permissionsLoading}
            message="You don't have permission to manage this canvas."
          >
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
                disabled={deleteCanvasMutation.isPending || updateCanvasGroupMembershipMutation.isPending}
              >
                <MoreVertical size={16} />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <PermissionTooltip
                allowed={canUpdateCanvases || permissionsLoading}
                message="You don't have permission to update canvases."
              >
                <DropdownMenuItem
                  onClick={(event: MouseEvent<HTMLElement>) => {
                    event.preventDefault();
                    event.stopPropagation();
                    if (!canUpdateCanvases) return;
                    onEdit(canvas);
                  }}
                  disabled={!canUpdateCanvases}
                >
                  <Pencil size={16} />
                  Change Name
                </DropdownMenuItem>
              </PermissionTooltip>

              <DropdownMenuSub>
                <DropdownMenuSubTrigger disabled={!canUpdateCanvases}>
                  <FolderPlus size={16} />
                  Add to Group
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent className="w-64">
                  {canvasGroups.length > 0 && (
                    <>
                      {canvasGroups.map((group) => (
                        <DropdownMenuItem
                          key={group.id}
                          onClick={() => handleAssignToGroup(group.id)}
                          disabled={group.id === canvas.canvasGroupId || updateCanvasGroupMembershipMutation.isPending}
                        >
                          <span className={cn("h-3 w-3 rounded-full", GROUP_SWATCH_CLASSES[group.backgroundColor])} />
                          <span className="truncate">{group.title}</span>
                          {group.id === canvas.canvasGroupId ? <Check className="ml-auto h-4 w-4" /> : null}
                        </DropdownMenuItem>
                      ))}

                      <DropdownMenuSeparator />
                    </>
                  )}

                  <form
                    className="space-y-3 p-3"
                    onSubmit={handleCreateGroup}
                    onClick={(event) => event.stopPropagation()}
                  >
                    <Input
                      value={newGroupTitle}
                      onChange={(event) => setNewGroupTitle(event.target.value)}
                      onKeyDown={(event) => event.stopPropagation()}
                      placeholder="New group name"
                      className="h-8"
                      maxLength={128}
                      disabled={!canUpdateCanvases || createCanvasGroupMutation.isPending}
                    />
                    <ColorSwatchPicker
                      selectedColor={newGroupColor}
                      onSelect={(color) => setNewGroupColor(color)}
                      size="sm"
                    />
                    <Button
                      type="submit"
                      size="sm"
                      className="w-full"
                      disabled={!newGroupTitle.trim() || createCanvasGroupMutation.isPending}
                    >
                      Create Group
                    </Button>
                  </form>
                </DropdownMenuSubContent>
              </DropdownMenuSub>

              {canvas.canvasGroupId ? (
                <DropdownMenuItem
                  onClick={handleRemoveFromGroup}
                  disabled={!canUpdateCanvases || updateCanvasGroupMembershipMutation.isPending}
                >
                  <FolderMinus size={16} />
                  Remove from Group
                </DropdownMenuItem>
              ) : null}

              <PermissionTooltip
                allowed={canDeleteCanvases || permissionsLoading}
                message="You don't have permission to delete canvases."
              >
                <DropdownMenuItem onClick={openDialog} disabled={!canDeleteCanvases}>
                  <Trash2 size={16} />
                  Delete Canvas
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
