import type { CanvasFoldersCanvasFolder } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/Input/input";
import { Text } from "@/components/Text/text";
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@/ui/dropdownMenu";
import {
  DEFAULT_CANVAS_FOLDER_COLOR,
  canvasKeys,
  useCreateCanvasFolder,
  useUpdateCanvasFolderMembership,
  type CanvasFolderColor,
} from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { Check, FolderMinus, FolderPlus } from "lucide-react";
import { useState, type FormEvent } from "react";
import { FOLDER_COLOR_OPTIONS } from "./canvasFolderStyles";
import { CanvasFolderColorPicker } from "./CanvasFolderColorPicker";
import type { CanvasCardData, CanvasFolderData } from "./types";

interface CanvasFolderSubmenuProps {
  canvas: CanvasCardData;
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  canUpdateCanvases: boolean;
}

export function CanvasFolderSubmenu({
  canvas,
  canvasFolders,
  organizationId,
  canUpdateCanvases,
}: CanvasFolderSubmenuProps) {
  const [newFolderTitle, setNewFolderTitle] = useState("");
  const [newFolderColor, setNewFolderColor] = useState<CanvasFolderColor>(DEFAULT_CANVAS_FOLDER_COLOR);
  const createCanvasFolderMutation = useCreateCanvasFolder(organizationId);
  const updateCanvasFolderMembershipMutation = useUpdateCanvasFolderMembership(organizationId);
  const queryClient = useQueryClient();
  const folderActionLabel = canvas.canvasFolderId ? "Move to Folder" : "Add to Folder";
  const isDuplicateNewFolderTitle = hasDuplicateFolderTitle(canvasFolders, newFolderTitle);

  const handleAssignToFolder = async (folderId: string) => {
    if (!canUpdateCanvases || folderId === canvas.canvasFolderId) return;

    const folder = canvasFolders.find((canvasFolder) => canvasFolder.id === folderId);
    if (!folder) {
      showErrorToast("Folder not found");
      return;
    }

    try {
      await updateCanvasFolderMembershipMutation.mutateAsync({
        folderId: folder.id,
        title: folder.title,
        backgroundColor: folder.backgroundColor,
        canvasIds: addCanvasToFolder(folder.canvasIds, canvas.id),
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to add canvas to folder"));
    }
  };

  const handleRemoveFromFolder = async () => {
    if (!canUpdateCanvases || !canvas.canvasFolderId) return;

    const folder = canvasFolders.find((canvasFolder) => canvasFolder.id === canvas.canvasFolderId);
    if (!folder) {
      showErrorToast("Folder not found");
      return;
    }

    try {
      await updateCanvasFolderMembershipMutation.mutateAsync({
        folderId: folder.id,
        title: folder.title,
        backgroundColor: folder.backgroundColor,
        canvasIds: removeCanvasFromFolder(folder.canvasIds, canvas.id),
      });
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

    let responseFolderId: string | undefined;
    try {
      const response = await createCanvasFolderMutation.mutateAsync({ title, backgroundColor: newFolderColor });
      responseFolderId = response.data?.folder?.metadata?.id;
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create folder"));
      return;
    }

    try {
      const folderId = await resolveCreatedFolderId(queryClient, organizationId, title, responseFolderId);

      await updateCanvasFolderMembershipMutation.mutateAsync({
        folderId,
        title,
        backgroundColor: newFolderColor,
        canvasIds: [canvas.id],
      });
    } catch (error) {
      setNewFolderTitle("");
      setNewFolderColor(DEFAULT_CANVAS_FOLDER_COLOR);
      showErrorToast(getFolderCreatedAssignmentErrorMessage(error));
      return;
    }

    setNewFolderTitle("");
    setNewFolderColor(DEFAULT_CANVAS_FOLDER_COLOR);
    showSuccessToast("Folder created");
  };

  return (
    <>
      <DropdownMenuSub>
        <DropdownMenuSubTrigger disabled={!canUpdateCanvases}>
          <FolderPlus size={16} />
          {folderActionLabel}
        </DropdownMenuSubTrigger>
        <DropdownMenuSubContent className="w-64">
          <CanvasFolderList
            canvas={canvas}
            canvasFolders={canvasFolders}
            isUpdatingMembership={updateCanvasFolderMembershipMutation.isPending}
            onAssignToFolder={(folderId) => void handleAssignToFolder(folderId)}
          />
          <CreateCanvasFolderForm
            title={newFolderTitle}
            backgroundColor={newFolderColor}
            isDuplicateTitle={isDuplicateNewFolderTitle}
            canUpdateCanvases={canUpdateCanvases}
            isCreatingFolder={createCanvasFolderMutation.isPending}
            onTitleChange={setNewFolderTitle}
            onColorChange={setNewFolderColor}
            onSubmit={handleCreateFolder}
          />
        </DropdownMenuSubContent>
      </DropdownMenuSub>

      <RemoveFromFolderItem
        canvasFolderId={canvas.canvasFolderId}
        canUpdateCanvases={canUpdateCanvases}
        isUpdatingMembership={updateCanvasFolderMembershipMutation.isPending}
        onRemoveFromFolder={() => void handleRemoveFromFolder()}
      />
    </>
  );
}

function hasDuplicateFolderTitle(canvasFolders: CanvasFolderData[], title: string) {
  const normalizedTitle = title.trim().toLowerCase();
  return canvasFolders.some((folder) => folder.title.trim().toLowerCase() === normalizedTitle);
}

async function resolveCreatedFolderId(
  queryClient: QueryClient,
  organizationId: string,
  title: string,
  responseFolderId?: string,
) {
  if (responseFolderId) {
    return responseFolderId;
  }

  await queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
  await queryClient.refetchQueries({ queryKey: canvasKeys.folderList(organizationId), type: "active" });

  const folders = queryClient.getQueryData<CanvasFoldersCanvasFolder[]>(canvasKeys.folderList(organizationId)) || [];
  const folderId =
    folders.find((folder) => folder.spec?.title?.trim().toLowerCase() === title.toLowerCase())?.metadata?.id || "";

  if (!folderId) {
    throw new Error("missing canvas folder id");
  }

  return folderId;
}

function addCanvasToFolder(canvasIds: string[], canvasId: string) {
  return canvasIds.includes(canvasId) ? canvasIds : [...canvasIds, canvasId];
}

function removeCanvasFromFolder(canvasIds: string[], canvasId: string) {
  return canvasIds.filter((id) => id !== canvasId);
}

function getFolderCreatedAssignmentErrorMessage(error: unknown) {
  const message = getApiErrorMessage(error, "failed to add canvas to it");
  return `Folder created, but ${message.charAt(0).toLowerCase()}${message.slice(1)}`;
}

function RemoveFromFolderItem({
  canvasFolderId,
  canUpdateCanvases,
  isUpdatingMembership,
  onRemoveFromFolder,
}: {
  canvasFolderId?: string;
  canUpdateCanvases: boolean;
  isUpdatingMembership: boolean;
  onRemoveFromFolder: () => void;
}) {
  if (!canvasFolderId) {
    return null;
  }

  return (
    <DropdownMenuItem onClick={onRemoveFromFolder} disabled={!canUpdateCanvases || isUpdatingMembership}>
      <FolderMinus size={16} />
      Remove from Folder
    </DropdownMenuItem>
  );
}

function CanvasFolderList({
  canvas,
  canvasFolders,
  isUpdatingMembership,
  onAssignToFolder,
}: {
  canvas: CanvasCardData;
  canvasFolders: CanvasFolderData[];
  isUpdatingMembership: boolean;
  onAssignToFolder: (folderId: string) => void;
}) {
  if (canvasFolders.length === 0) {
    return null;
  }

  return (
    <>
      {canvasFolders.map((folder) => (
        <DropdownMenuItem
          key={folder.id}
          onClick={() => onAssignToFolder(folder.id)}
          disabled={folder.id === canvas.canvasFolderId || isUpdatingMembership}
        >
          <span className={cn("h-3 w-3 rounded-full", FOLDER_COLOR_OPTIONS[folder.backgroundColor].swatchClass)} />
          <span className="truncate">{folder.title}</span>
          {folder.id === canvas.canvasFolderId ? <Check className="ml-auto h-4 w-4" /> : null}
        </DropdownMenuItem>
      ))}

      <DropdownMenuSeparator />
    </>
  );
}

function CreateCanvasFolderForm({
  title,
  backgroundColor,
  isDuplicateTitle,
  canUpdateCanvases,
  isCreatingFolder,
  onTitleChange,
  onColorChange,
  onSubmit,
}: {
  title: string;
  backgroundColor: CanvasFolderColor;
  isDuplicateTitle: boolean;
  canUpdateCanvases: boolean;
  isCreatingFolder: boolean;
  onTitleChange: (title: string) => void;
  onColorChange: (color: CanvasFolderColor) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="space-y-3 p-3" onSubmit={onSubmit} onClick={(event) => event.stopPropagation()}>
      <Input
        value={title}
        onChange={(event) => onTitleChange(event.target.value)}
        onKeyDown={(event) => event.stopPropagation()}
        placeholder="New folder name"
        className="h-8"
        maxLength={128}
        disabled={!canUpdateCanvases || isCreatingFolder}
      />
      {isDuplicateTitle ? (
        <Text className="text-xs text-red-600 dark:text-red-300">Folder name already exists</Text>
      ) : null}
      <CanvasFolderColorPicker selectedColor={backgroundColor} onColorChange={onColorChange} />
      <Button
        type="submit"
        size="sm"
        className="w-full"
        disabled={!title.trim() || isDuplicateTitle || isCreatingFolder}
      >
        Create Folder
      </Button>
    </form>
  );
}
