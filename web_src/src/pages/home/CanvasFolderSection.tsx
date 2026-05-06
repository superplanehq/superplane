import { PermissionTooltip } from "@/components/PermissionGate";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Heading } from "@/components/Heading/heading";
import { Input } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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
  useUpdateCanvasFolder,
  type CanvasFolderColor,
} from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { ArrowDown, ArrowUp, Check, FolderOpen, MoreVertical, Palette, Pencil, Trash2 } from "lucide-react";
import { useEffect, useRef, useState, type KeyboardEvent, type MutableRefObject, type RefObject } from "react";
import { CanvasCardsGrid } from "./CanvasCardsGrid";
import { FOLDER_COLOR_OPTIONS } from "./canvasFolderStyles";
import type { CanvasCardData, CanvasFolderData } from "./types";

interface CanvasFolderSectionProps {
  folder: CanvasFolderData;
  canvases: CanvasCardData[];
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
}

interface CanvasFolderTitleProps {
  folder: CanvasFolderData;
  canUpdateCanvases: boolean;
  renameInputRef: RefObject<HTMLInputElement | null>;
  isRenaming: boolean;
  draftTitle: string;
  isPending: boolean;
  onDraftTitleChange: (title: string) => void;
  onStartRenaming: () => void;
  onSubmitRename: () => void;
  onCancelRenaming: () => void;
  onFocusRenameInput: () => void;
  isSubmittingRenameRef: MutableRefObject<boolean>;
  ignoreBlurUntilRef: MutableRefObject<number>;
}

function CanvasFolderTitle({
  folder,
  canUpdateCanvases,
  renameInputRef,
  isRenaming,
  draftTitle,
  isPending,
  onDraftTitleChange,
  onStartRenaming,
  onSubmitRename,
  onCancelRenaming,
  onFocusRenameInput,
  isSubmittingRenameRef,
  ignoreBlurUntilRef,
}: CanvasFolderTitleProps) {
  const handleRenameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      onSubmitRename();
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      onCancelRenaming();
    }
  };

  if (!canUpdateCanvases) {
    return (
      <Heading level={3} className="mb-0 truncate !text-base font-medium text-white">
        {folder.title}
      </Heading>
    );
  }

  if (isRenaming) {
    return (
      <Input
        ref={renameInputRef}
        value={draftTitle}
        onChange={(event) => onDraftTitleChange(event.target.value)}
        onBlur={() => {
          if (ignoreBlurUntilRef.current > Date.now()) {
            onFocusRenameInput();
            return;
          }

          if (!isSubmittingRenameRef.current) {
            onSubmitRename();
          }
        }}
        onKeyDown={handleRenameKeyDown}
        aria-label="Folder name"
        maxLength={128}
        disabled={isPending}
        className="h-6 max-w-[320px] border-white/50 bg-white/5 px-1 text-base font-medium text-white shadow-none placeholder:text-white/60 focus-visible:border-white/60"
      />
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={onStartRenaming}
          className="flex h-6 max-w-xl items-center rounded-md border border-transparent px-1 text-left transition hover:border-white/25 hover:bg-white/5"
          aria-label={`Rename folder ${folder.title}`}
        >
          <span className="truncate text-base font-medium text-white">{folder.title}</span>
        </button>
      </TooltipTrigger>
      <TooltipContent>Rename</TooltipContent>
    </Tooltip>
  );
}

export function CanvasFolderSection({
  folder,
  canvases,
  canvasFolders,
  organizationId,
  onEditCanvas,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
  canMoveUp,
  canMoveDown,
}: CanvasFolderSectionProps) {
  const [draftTitle, setDraftTitle] = useState(folder.title);
  const [isRenaming, setIsRenaming] = useState(false);
  const renameInputRef = useRef<HTMLInputElement>(null);
  const isSubmittingRenameRef = useRef(false);
  const ignoreBlurUntilRef = useRef(0);
  const updateCanvasFolderMutation = useUpdateCanvasFolder(organizationId);

  useEffect(() => {
    if (!isRenaming) {
      setDraftTitle(folder.title);
    }
  }, [folder.title, isRenaming]);

  const focusRenameInput = (selectText = false) => {
    window.setTimeout(() => {
      renameInputRef.current?.focus();
      if (selectText) {
        renameInputRef.current?.select();
      }
    }, 0);
  };

  const startRenaming = ({ preserveFocus = false }: { preserveFocus?: boolean } = {}) => {
    if (!canUpdateCanvases || updateCanvasFolderMutation.isPending) return;

    if (preserveFocus) {
      ignoreBlurUntilRef.current = Date.now() + 200;
    }

    setIsRenaming(true);
    focusRenameInput(true);
  };

  const cancelRenaming = () => {
    setDraftTitle(folder.title);
    setIsRenaming(false);
  };

  const submitRename = async () => {
    if (!canUpdateCanvases || isSubmittingRenameRef.current) return;

    const title = draftTitle.trim();
    if (!title) {
      showErrorToast("Folder name is required");
      focusRenameInput();
      return;
    }

    if (title === folder.title) {
      cancelRenaming();
      return;
    }

    isSubmittingRenameRef.current = true;

    try {
      await updateCanvasFolderMutation.mutateAsync({
        folderId: folder.id,
        title,
        backgroundColor: folder.backgroundColor,
      });
      setIsRenaming(false);
      showSuccessToast("Folder renamed");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename folder"));
      focusRenameInput();
    } finally {
      isSubmittingRenameRef.current = false;
    }
  };

  return (
    <section className={cn("w-full rounded-md p-4", FOLDER_COLOR_OPTIONS[folder.backgroundColor].backgroundClass)}>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          <CanvasFolderTitle
            folder={folder}
            canUpdateCanvases={canUpdateCanvases}
            renameInputRef={renameInputRef}
            isRenaming={isRenaming}
            draftTitle={draftTitle}
            isPending={updateCanvasFolderMutation.isPending}
            onDraftTitleChange={setDraftTitle}
            onStartRenaming={() => startRenaming()}
            onSubmitRename={() => void submitRename()}
            onCancelRenaming={cancelRenaming}
            onFocusRenameInput={focusRenameInput}
            isSubmittingRenameRef={isSubmittingRenameRef}
            ignoreBlurUntilRef={ignoreBlurUntilRef}
          />
        </div>
        <CanvasFolderActionsMenu
          folder={folder}
          organizationId={organizationId}
          canUpdateCanvases={canUpdateCanvases}
          permissionsLoading={permissionsLoading}
          canMoveUp={canMoveUp}
          canMoveDown={canMoveDown}
          onRenameRequest={() => startRenaming({ preserveFocus: true })}
        />
      </div>

      {canvases.length > 0 ? (
        <CanvasCardsGrid
          canvases={canvases}
          canvasFolders={canvasFolders}
          organizationId={organizationId}
          onEditCanvas={onEditCanvas}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
        />
      ) : (
        <div className="flex min-h-40 flex-col items-center justify-center gap-2 rounded-md px-4 py-8 text-center text-[13px] font-medium text-white/80">
          <FolderOpen size={18} className="text-white/80" />
          <span>No canvases in this folder</span>
        </div>
      )}
    </section>
  );
}

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

function CanvasFolderActionsMenu({
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
  const moveCanvasFolderMutation = useUpdateCanvasFolder(organizationId);
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
