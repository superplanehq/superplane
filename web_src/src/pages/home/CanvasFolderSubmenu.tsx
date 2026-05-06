import type { CanvasesCanvasFolder } from "@/api-client";
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
  CANVAS_FOLDER_COLORS,
  DEFAULT_CANVAS_FOLDER_COLOR,
  canvasKeys,
  useCreateCanvasFolder,
  useUpdateCanvasFolderMembership,
  type CanvasFolderColor,
} from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { Check, FolderMinus, FolderPlus } from "lucide-react";
import { useState, type FormEvent } from "react";
import { FOLDER_COLOR_OPTIONS } from "./canvasFolderStyles";
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
  const normalizedNewFolderTitle = newFolderTitle.trim().toLowerCase();
  const isDuplicateNewFolderTitle = canvasFolders.some(
    (folder) => folder.title.trim().toLowerCase() === normalizedNewFolderTitle,
  );

  const handleAssignToFolder = async (folderId: string) => {
    if (!canUpdateCanvases || folderId === canvas.canvasFolderId) return;

    try {
      await updateCanvasFolderMembershipMutation.mutateAsync({ canvasId: canvas.id, folderId });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to add canvas to folder"));
    }
  };

  const handleRemoveFromFolder = async () => {
    if (!canUpdateCanvases || !canvas.canvasFolderId) return;

    try {
      await updateCanvasFolderMembershipMutation.mutateAsync({ canvasId: canvas.id });
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
      const folderId = await resolveCreatedFolderId(title, response.data?.folder?.metadata?.id);

      await updateCanvasFolderMembershipMutation.mutateAsync({ canvasId: canvas.id, folderId });

      setNewFolderTitle("");
      setNewFolderColor(DEFAULT_CANVAS_FOLDER_COLOR);
      showSuccessToast("Folder created");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create folder"));
    }
  };

  const resolveCreatedFolderId = async (title: string, responseFolderId?: string) => {
    if (responseFolderId) {
      return responseFolderId;
    }

    await queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
    await queryClient.refetchQueries({ queryKey: canvasKeys.folderList(organizationId), type: "active" });

    const folders = queryClient.getQueryData<CanvasesCanvasFolder[]>(canvasKeys.folderList(organizationId)) || [];
    const folderId =
      folders.find((folder) => folder.spec?.title?.trim().toLowerCase() === title.toLowerCase())?.metadata?.id || "";

    if (!folderId) {
      throw new Error("missing canvas folder id");
    }

    return folderId;
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

      {canvas.canvasFolderId ? (
        <DropdownMenuItem
          onClick={handleRemoveFromFolder}
          disabled={!canUpdateCanvases || updateCanvasFolderMembershipMutation.isPending}
        >
          <FolderMinus size={16} />
          Remove from Folder
        </DropdownMenuItem>
      ) : null}
    </>
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

function CanvasFolderColorPicker({
  selectedColor,
  onColorChange,
}: {
  selectedColor: CanvasFolderColor;
  onColorChange: (color: CanvasFolderColor) => void;
}) {
  return (
    <div className="flex items-center gap-2">
      {CANVAS_FOLDER_COLORS.map((color) => (
        <button
          key={color}
          type="button"
          aria-label={`${FOLDER_COLOR_OPTIONS[color].label} folder color`}
          className={cn(
            "flex h-5 w-5 items-center justify-center rounded-full border border-slate-950/15 text-white",
            FOLDER_COLOR_OPTIONS[color].swatchClass,
            selectedColor === color && "ring-2 ring-gray-900 ring-offset-1",
          )}
          onClick={() => onColorChange(color)}
        >
          {selectedColor === color ? <Check className="h-3 w-3" /> : null}
        </button>
      ))}
    </div>
  );
}
