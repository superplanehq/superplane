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
import { Input } from "@/components/Input/input";
import { Text } from "@/components/Text/text";
import {
  CANVAS_FOLDER_COLORS,
  DEFAULT_CANVAS_FOLDER_COLOR,
  canvasKeys,
  useCreateCanvasFolder,
  useDeleteCanvas,
  useUpdateCanvasFolder,
  type CanvasFolderColor,
} from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import type { CanvasesCanvasFolder } from "@/api-client";
import { Check, FolderMinus, FolderPlus, MoreVertical, Pencil, Trash2 } from "lucide-react";
import { useState, type FormEvent, type MouseEvent } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { FOLDER_COLOR_OPTIONS } from "./canvasFolderStyles";
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
  const [newFolderTitle, setNewFolderTitle] = useState("");
  const [newFolderColor, setNewFolderColor] = useState<CanvasFolderColor>(DEFAULT_CANVAS_FOLDER_COLOR);
  const deleteCanvasMutation = useDeleteCanvas(organizationId);
  const createCanvasFolderMutation = useCreateCanvasFolder(organizationId);
  const updateCanvasFolderMutation = useUpdateCanvasFolder(organizationId);
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const canManage = canUpdateCanvases || canDeleteCanvases;
  const folderActionLabel = canvas.canvasFolderId ? "Move to Folder" : "Add to Folder";
  const normalizedNewFolderTitle = newFolderTitle.trim().toLowerCase();
  const isDuplicateNewFolderTitle = canvasFolders.some(
    (folder) => folder.title.trim().toLowerCase() === normalizedNewFolderTitle,
  );

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const openDialog = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDialogOpen(true);
  };

  const handleAssignToFolder = async (folderId: string) => {
    if (!canUpdateCanvases || folderId === canvas.canvasFolderId) return;

    try {
      await updateCanvasFolderMutation.mutateAsync({ canvasId: canvas.id, targetFolderId: folderId });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to add canvas to folder"));
    }
  };

  const handleRemoveFromFolder = async () => {
    if (!canUpdateCanvases || !canvas.canvasFolderId) return;

    try {
      await updateCanvasFolderMutation.mutateAsync({ canvasId: canvas.id });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to remove canvas from folder"));
    }
  };

  const handleCreateFolder = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!canUpdateCanvases) return;

    const title = newFolderTitle.trim();
    if (!title) return;

    if (isDuplicateNewFolderTitle) {
      showErrorToast("Folder name already exists");
      return;
    }

    try {
      const response = await createCanvasFolderMutation.mutateAsync({ title, backgroundColor: newFolderColor });
      let folderId = response.data?.folder?.metadata?.id;

      if (!folderId) {
        await queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
        await queryClient.refetchQueries({ queryKey: canvasKeys.folderList(organizationId), type: "active" });

        const folders = queryClient.getQueryData<CanvasesCanvasFolder[]>(canvasKeys.folderList(organizationId)) || [];
        folderId =
          folders.find((folder) => folder.spec?.title?.trim().toLowerCase() === title.toLowerCase())?.metadata?.id ||
          "";
      }

      if (!folderId) {
        throw new Error("missing canvas folder id");
      }

      await updateCanvasFolderMutation.mutateAsync({ canvasId: canvas.id, targetFolderId: folderId });

      setNewFolderTitle("");
      setNewFolderColor(DEFAULT_CANVAS_FOLDER_COLOR);
      showSuccessToast("Folder created");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create folder"));
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
                disabled={deleteCanvasMutation.isPending || updateCanvasFolderMutation.isPending}
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
                  {folderActionLabel}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent className="w-64">
                  {canvasFolders.length > 0 && (
                    <>
                      {canvasFolders.map((folder) => (
                        <DropdownMenuItem
                          key={folder.id}
                          onClick={() => handleAssignToFolder(folder.id)}
                          disabled={folder.id === canvas.canvasFolderId || updateCanvasFolderMutation.isPending}
                        >
                          <span
                            className={cn(
                              "h-3 w-3 rounded-full",
                              FOLDER_COLOR_OPTIONS[folder.backgroundColor].swatchClass,
                            )}
                          />
                          <span className="truncate">{folder.title}</span>
                          {folder.id === canvas.canvasFolderId ? <Check className="ml-auto h-4 w-4" /> : null}
                        </DropdownMenuItem>
                      ))}

                      <DropdownMenuSeparator />
                    </>
                  )}

                  <form
                    className="space-y-3 p-3"
                    onSubmit={handleCreateFolder}
                    onClick={(event) => event.stopPropagation()}
                  >
                    <Input
                      value={newFolderTitle}
                      onChange={(event) => setNewFolderTitle(event.target.value)}
                      onKeyDown={(event) => event.stopPropagation()}
                      placeholder="New folder name"
                      className="h-8"
                      maxLength={128}
                      disabled={!canUpdateCanvases || createCanvasFolderMutation.isPending}
                    />
                    {isDuplicateNewFolderTitle ? (
                      <Text className="text-xs text-red-600 dark:text-red-300">Folder name already exists</Text>
                    ) : null}
                    <div className="flex items-center gap-2">
                      {CANVAS_FOLDER_COLORS.map((color) => (
                        <button
                          key={color}
                          type="button"
                          aria-label={`${FOLDER_COLOR_OPTIONS[color].label} folder color`}
                          className={cn(
                            "flex h-5 w-5 items-center justify-center rounded-full border border-slate-950/15 text-white",
                            FOLDER_COLOR_OPTIONS[color].swatchClass,
                            newFolderColor === color && "ring-2 ring-gray-900 ring-offset-1",
                          )}
                          onClick={() => setNewFolderColor(color)}
                        >
                          {newFolderColor === color ? <Check className="h-3 w-3" /> : null}
                        </button>
                      ))}
                    </div>
                    <Button
                      type="submit"
                      size="sm"
                      className="w-full"
                      disabled={
                        !newFolderTitle.trim() || isDuplicateNewFolderTitle || createCanvasFolderMutation.isPending
                      }
                    >
                      Create Folder
                    </Button>
                  </form>
                </DropdownMenuSubContent>
              </DropdownMenuSub>

              {canvas.canvasFolderId ? (
                <DropdownMenuItem
                  onClick={handleRemoveFromFolder}
                  disabled={!canUpdateCanvases || updateCanvasFolderMutation.isPending}
                >
                  <FolderMinus size={16} />
                  Remove from Folder
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
