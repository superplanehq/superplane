import type { CanvasFoldersCanvasFolder } from "@/api-client";
import { LoadingButton } from "@/components/ui/loading-button";
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

type CreateCanvasFolderMutation = ReturnType<typeof useCreateCanvasFolder>;
type UpdateCanvasFolderMembershipMutation = ReturnType<typeof useUpdateCanvasFolderMembership>;

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
  const isCreatingFolder = createCanvasFolderMutation.isPending;
  const isUpdatingMembership = updateCanvasFolderMembershipMutation.isPending;

  const resetNewFolderForm = () => {
    setNewFolderTitle("");
    setNewFolderColor(DEFAULT_CANVAS_FOLDER_COLOR);
  };

  const handleAssignToFolder = (folderId: string) =>
    assignCanvasToFolder({
      canvas,
      canvasFolders,
      folderId,
      canUpdateCanvases,
      updateCanvasFolderMembershipMutation,
    });

  const handleRemoveFromFolder = () =>
    removeCanvasFromFolder({
      canvas,
      canvasFolders,
      canUpdateCanvases,
      updateCanvasFolderMembershipMutation,
    });

  const handleCreateFolder = (event: FormEvent<HTMLFormElement>) =>
    createFolderAndAssignCanvas({
      event,
      canvasId: canvas.id,
      title: newFolderTitle,
      backgroundColor: newFolderColor,
      organizationId,
      queryClient,
      canUpdateCanvases,
      isDuplicateTitle: isDuplicateNewFolderTitle,
      createCanvasFolderMutation,
      updateCanvasFolderMembershipMutation,
      onReset: resetNewFolderForm,
    });

  return (
    <>
      <DropdownMenuSub>
        <DropdownMenuSubTrigger disabled={!canUpdateCanvases || isCreatingFolder || isUpdatingMembership}>
          <FolderPlus size={16} />
          {folderActionLabel}
        </DropdownMenuSubTrigger>
        <DropdownMenuSubContent className="w-64">
          <CanvasFolderList
            canvas={canvas}
            canvasFolders={canvasFolders}
            isUpdatingMembership={isUpdatingMembership}
            onAssignToFolder={(folderId) => void handleAssignToFolder(folderId)}
          />
          <CreateCanvasFolderForm
            title={newFolderTitle}
            backgroundColor={newFolderColor}
            isDuplicateTitle={isDuplicateNewFolderTitle}
            canUpdateCanvases={canUpdateCanvases}
            isCreatingFolder={isCreatingFolder}
            isUpdatingMembership={isUpdatingMembership}
            onTitleChange={setNewFolderTitle}
            onColorChange={setNewFolderColor}
            onSubmit={handleCreateFolder}
          />
        </DropdownMenuSubContent>
      </DropdownMenuSub>

      <RemoveFromFolderItem
        canvasFolderId={canvas.canvasFolderId}
        canUpdateCanvases={canUpdateCanvases}
        isUpdatingMembership={isUpdatingMembership}
        onRemoveFromFolder={() => void handleRemoveFromFolder()}
      />
    </>
  );
}

function hasDuplicateFolderTitle(canvasFolders: CanvasFolderData[], title: string) {
  const normalizedTitle = title.trim().toLowerCase();
  return canvasFolders.some((folder) => folder.title.trim().toLowerCase() === normalizedTitle);
}

async function assignCanvasToFolder({
  canvas,
  canvasFolders,
  folderId,
  canUpdateCanvases,
  updateCanvasFolderMembershipMutation,
}: {
  canvas: CanvasCardData;
  canvasFolders: CanvasFolderData[];
  folderId: string;
  canUpdateCanvases: boolean;
  updateCanvasFolderMembershipMutation: UpdateCanvasFolderMembershipMutation;
}) {
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
}

async function removeCanvasFromFolder({
  canvas,
  canvasFolders,
  canUpdateCanvases,
  updateCanvasFolderMembershipMutation,
}: {
  canvas: CanvasCardData;
  canvasFolders: CanvasFolderData[];
  canUpdateCanvases: boolean;
  updateCanvasFolderMembershipMutation: UpdateCanvasFolderMembershipMutation;
}) {
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
      canvasIds: removeCanvasId(folder.canvasIds, canvas.id),
    });
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to remove canvas from folder"));
  }
}

async function createFolderAndAssignCanvas({
  event,
  canvasId,
  title,
  backgroundColor,
  organizationId,
  queryClient,
  canUpdateCanvases,
  isDuplicateTitle,
  createCanvasFolderMutation,
  updateCanvasFolderMembershipMutation,
  onReset,
}: {
  event: FormEvent<HTMLFormElement>;
  canvasId: string;
  title: string;
  backgroundColor: CanvasFolderColor;
  organizationId: string;
  queryClient: QueryClient;
  canUpdateCanvases: boolean;
  isDuplicateTitle: boolean;
  createCanvasFolderMutation: CreateCanvasFolderMutation;
  updateCanvasFolderMembershipMutation: UpdateCanvasFolderMembershipMutation;
  onReset: () => void;
}) {
  event.preventDefault();
  event.stopPropagation();
  const trimmedTitle = title.trim();
  if (!canUpdateCanvases || !trimmedTitle) return;

  if (isDuplicateTitle) {
    showErrorToast("Folder name already exists");
    return;
  }

  const createdFolder = await createCanvasFolder(trimmedTitle, backgroundColor, createCanvasFolderMutation);
  if (!createdFolder.ok) return;

  try {
    const folderId = await resolveCreatedFolderId(queryClient, organizationId, trimmedTitle, createdFolder.folderId);
    await updateCanvasFolderMembershipMutation.mutateAsync({
      folderId,
      title: trimmedTitle,
      backgroundColor,
      canvasIds: [canvasId],
    });
  } catch (error) {
    onReset();
    showErrorToast(getFolderCreatedAssignmentErrorMessage(error));
    return;
  }

  onReset();
  showSuccessToast("Folder created");
}

async function createCanvasFolder(
  title: string,
  backgroundColor: CanvasFolderColor,
  createCanvasFolderMutation: CreateCanvasFolderMutation,
) {
  try {
    const response = await createCanvasFolderMutation.mutateAsync({ title, backgroundColor });
    return { ok: true as const, folderId: response.data?.folder?.metadata?.id };
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to create folder"));
    return { ok: false as const };
  }
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

function removeCanvasId(canvasIds: string[], canvasId: string) {
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
  isUpdatingMembership,
  onTitleChange,
  onColorChange,
  onSubmit,
}: {
  title: string;
  backgroundColor: CanvasFolderColor;
  isDuplicateTitle: boolean;
  canUpdateCanvases: boolean;
  isCreatingFolder: boolean;
  isUpdatingMembership: boolean;
  onTitleChange: (title: string) => void;
  onColorChange: (color: CanvasFolderColor) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  const isSaving = isCreatingFolder || isUpdatingMembership;
  const loadingText = isCreatingFolder ? "Creating..." : "Adding...";

  return (
    <form className="space-y-3 p-3" onSubmit={onSubmit} onClick={(event) => event.stopPropagation()}>
      <Input
        value={title}
        onChange={(event) => onTitleChange(event.target.value)}
        onKeyDown={(event) => event.stopPropagation()}
        placeholder="New folder name"
        className="h-8"
        maxLength={128}
        disabled={!canUpdateCanvases || isSaving}
      />
      {isDuplicateTitle ? (
        <Text className="text-xs text-red-600 dark:text-red-300">Folder name already exists</Text>
      ) : null}
      <CanvasFolderColorPicker
        selectedColor={backgroundColor}
        onColorChange={onColorChange}
        isColorDisabled={() => isSaving}
      />
      <LoadingButton
        type="submit"
        size="sm"
        className="w-full"
        disabled={!title.trim() || isDuplicateTitle || isSaving}
        loading={isSaving}
        loadingText={loadingText}
      >
        Create Folder
      </LoadingButton>
    </form>
  );
}
