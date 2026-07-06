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
import { FOLDER_COLOR_OPTIONS, CANVAS_FOLDER_SECTION_SHELL_CLASS } from "./canvasFolderStyles";
import type { CanvasCardData, CanvasFolderData } from "./types";

interface CanvasFolderSectionProps {
  folder: CanvasFolderData;
  canvases: CanvasCardData[];
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  onTogglePin: (canvasId: string, pinned: boolean) => void;
  onToggleStar: (canvasId: string, starred: boolean) => void;
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
  onTogglePin,
  onToggleStar,
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
          onTogglePin={onTogglePin}
          onToggleStar={onToggleStar}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
        />
      ) : (
        <EmptyCanvasFolder />
      )}
    </section>
  );
}

function EmptyCanvasFolder() {
  return (
    <div className="flex min-h-40 flex-col items-center justify-center gap-2 rounded-md px-4 py-8 text-center text-[13px] font-medium text-white/80">
      <FolderOpen size={18} className="text-white/80" />
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
