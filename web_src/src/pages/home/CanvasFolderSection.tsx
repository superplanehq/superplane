import { Heading } from "@/components/Heading/heading";
import { Input } from "@/components/Input/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useUpdateCanvasFolder } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { FolderOpen } from "lucide-react";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type MutableRefObject,
  type RefObject,
} from "react";
import { CanvasFolderActionsMenu } from "./CanvasFolderActionsMenu";
import { CanvasCardsGrid } from "./CanvasCardsGrid";
import { FOLDER_COLOR_OPTIONS, CANVAS_FOLDER_SECTION_SHELL_CLASS, folderColorStyles } from "./canvasFolderStyles";
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

interface SubmitCanvasFolderRenameOptions {
  canUpdateCanvases: boolean;
  isSubmittingRenameRef: MutableRefObject<boolean>;
  draftTitle: string;
  folder: CanvasFolderData;
  focusRenameInput: () => void;
  cancelRenaming: () => void;
  updateFolder: (data: {
    folderId: string;
    title: string;
    backgroundColor: CanvasFolderData["backgroundColor"];
  }) => Promise<unknown>;
  setIsRenaming: (isRenaming: boolean) => void;
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
  const colorStyles = folderColorStyles(folder.backgroundColor);

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
      <Heading level={3} className={cn("mb-0 truncate !text-base font-medium", colorStyles.foregroundClass)}>
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
        className={cn("h-6 max-w-[320px] px-1 text-base font-medium shadow-none", colorStyles.renameInputClass)}
      />
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={onStartRenaming}
          className={cn(
            "flex h-6 max-w-xl items-center rounded-md border border-transparent px-1 text-left transition",
            colorStyles.headerInteractiveClass,
          )}
          aria-label={`Rename folder ${folder.title}`}
        >
          <span className={cn("truncate text-base font-medium", colorStyles.foregroundClass)}>{folder.title}</span>
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

  const focusRenameInput = useCallback((selectText = false) => {
    window.setTimeout(() => {
      renameInputRef.current?.focus();
      if (selectText) {
        renameInputRef.current?.select();
      }
    }, 0);
  }, []);

  const startRenaming = useCallback(
    ({ preserveFocus = false }: { preserveFocus?: boolean } = {}) => {
      if (!canUpdateCanvases || updateCanvasFolderMutation.isPending) return;

      if (preserveFocus) {
        ignoreBlurUntilRef.current = Date.now() + 200;
      }

      setIsRenaming(true);
      focusRenameInput(true);
    },
    [canUpdateCanvases, focusRenameInput, updateCanvasFolderMutation.isPending],
  );

  const startRenamingPreservingFocus = useCallback(() => {
    startRenaming({ preserveFocus: true });
  }, [startRenaming]);

  const cancelRenaming = () => {
    setDraftTitle(folder.title);
    setIsRenaming(false);
  };

  const submitRename = () =>
    submitCanvasFolderRename({
      canUpdateCanvases,
      isSubmittingRenameRef,
      draftTitle,
      folder,
      focusRenameInput,
      cancelRenaming,
      updateFolder: updateCanvasFolderMutation.mutateAsync,
      setIsRenaming,
    });

  return (
    <section
      className={cn(CANVAS_FOLDER_SECTION_SHELL_CLASS, FOLDER_COLOR_OPTIONS[folder.backgroundColor].backgroundClass)}
    >
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
          updateCanvasFolderMutation={updateCanvasFolderMutation}
          onRenameRequest={startRenamingPreservingFocus}
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
        <EmptyCanvasFolder backgroundColor={folder.backgroundColor} />
      )}
    </section>
  );
}

function EmptyCanvasFolder({ backgroundColor }: { backgroundColor: CanvasFolderData["backgroundColor"] }) {
  const colorStyles = folderColorStyles(backgroundColor);

  return (
    <div
      className={cn(
        "flex min-h-40 flex-col items-center justify-center gap-2 rounded-md px-4 py-8 text-center text-[13px] font-medium",
        colorStyles.foregroundMutedClass,
      )}
    >
      <FolderOpen size={18} className={colorStyles.foregroundMutedClass} />
      <span>No canvases in this folder</span>
    </div>
  );
}

async function submitCanvasFolderRename({
  canUpdateCanvases,
  isSubmittingRenameRef,
  draftTitle,
  folder,
  focusRenameInput,
  cancelRenaming,
  updateFolder,
  setIsRenaming,
}: SubmitCanvasFolderRenameOptions) {
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
    await updateFolder({
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
}
