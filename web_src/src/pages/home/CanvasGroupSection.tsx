import { FolderOpen } from "lucide-react";
import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Heading } from "../../components/Heading/heading";
import { Input } from "../../components/Input/input";
import { useUpdateCanvasGroup } from "../../hooks/useCanvasData";
import { cn } from "../../lib/utils";
import { showErrorToast, showSuccessToast } from "../../lib/toast";
import { getApiErrorMessage } from "../../lib/errors";
import { CanvasCard } from "./CanvasCard";
import { CanvasGroupActionsMenu } from "./CanvasGroupActionsMenu";
import { GROUP_BACKGROUND_CLASSES, type CanvasCardData, type CanvasGroupData } from "./shared";

interface CanvasGroupSectionProps {
  group: CanvasGroupData;
  canvases: CanvasCardData[];
  canvasGroups: CanvasGroupData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
}

export function CanvasGroupSection({
  group,
  canvases,
  canvasGroups,
  organizationId,
  onEditCanvas,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
  canMoveUp,
  canMoveDown,
}: CanvasGroupSectionProps) {
  const [draftTitle, setDraftTitle] = useState(group.title);
  const [isRenaming, setIsRenaming] = useState(false);
  const renameInputRef = useRef<HTMLInputElement>(null);
  const isSubmittingRenameRef = useRef(false);
  const ignoreBlurUntilRef = useRef(0);
  const updateCanvasGroupMutation = useUpdateCanvasGroup(organizationId);

  useEffect(() => {
    if (!isRenaming) {
      setDraftTitle(group.title);
    }
  }, [group.title, isRenaming]);

  const focusRenameInput = (selectText = false) => {
    window.setTimeout(() => {
      renameInputRef.current?.focus();
      if (selectText) {
        renameInputRef.current?.select();
      }
    }, 0);
  };

  const startRenaming = ({ preserveFocus = false }: { preserveFocus?: boolean } = {}) => {
    if (!canUpdateCanvases || updateCanvasGroupMutation.isPending) return;

    if (preserveFocus) {
      ignoreBlurUntilRef.current = Date.now() + 200;
    }

    setIsRenaming(true);
    focusRenameInput(true);
  };

  const cancelRenaming = () => {
    setDraftTitle(group.title);
    setIsRenaming(false);
  };

  const submitRename = async () => {
    if (!canUpdateCanvases || isSubmittingRenameRef.current) return;

    const title = draftTitle.trim();
    if (!title) {
      showErrorToast("Group name is required");
      focusRenameInput();
      return;
    }

    if (title === group.title) {
      cancelRenaming();
      return;
    }

    isSubmittingRenameRef.current = true;

    try {
      await updateCanvasGroupMutation.mutateAsync({
        groupId: group.id,
        title,
        backgroundColor: group.backgroundColor,
      });
      setIsRenaming(false);
      showSuccessToast("Group renamed");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename group"));
      focusRenameInput();
    } finally {
      isSubmittingRenameRef.current = false;
    }
  };

  const handleRenameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      void submitRename();
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      cancelRenaming();
    }
  };

  return (
    <section className={cn("w-full rounded-md p-4", GROUP_BACKGROUND_CLASSES[group.backgroundColor])}>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          {canUpdateCanvases ? (
            isRenaming ? (
              <Input
                ref={renameInputRef}
                value={draftTitle}
                onChange={(event) => setDraftTitle(event.target.value)}
                onBlur={() => {
                  if (ignoreBlurUntilRef.current > Date.now()) {
                    focusRenameInput();
                    return;
                  }

                  if (!isSubmittingRenameRef.current) {
                    void submitRename();
                  }
                }}
                onKeyDown={handleRenameKeyDown}
                aria-label="Group name"
                maxLength={128}
                disabled={updateCanvasGroupMutation.isPending}
                className="h-6 max-w-[320px] border-white/50 bg-white/5 px-1 text-base font-medium text-white shadow-none placeholder:text-white/60 focus-visible:border-white/60"
              />
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={() => startRenaming()}
                    className="flex h-6 max-w-xl items-center rounded-md border border-transparent px-1 text-left transition hover:border-white/25 hover:bg-white/5"
                    aria-label={`Rename group ${group.title}`}
                  >
                    <span className="truncate text-base font-medium text-white">{group.title}</span>
                  </button>
                </TooltipTrigger>
                <TooltipContent>Rename</TooltipContent>
              </Tooltip>
            )
          ) : (
            <Heading level={3} className="mb-0 truncate !text-base font-medium text-white">
              {group.title}
            </Heading>
          )}
        </div>
        <CanvasGroupActionsMenu
          group={group}
          organizationId={organizationId}
          canUpdateCanvases={canUpdateCanvases}
          permissionsLoading={permissionsLoading}
          canMoveUp={canMoveUp}
          canMoveDown={canMoveDown}
          onRenameRequest={() => startRenaming({ preserveFocus: true })}
        />
      </div>

      {canvases.length > 0 ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          {canvases.map((canvas) => (
            <CanvasCard
              key={canvas.id}
              canvas={canvas}
              canvasGroups={canvasGroups}
              organizationId={organizationId}
              onEdit={onEditCanvas}
              canUpdateCanvases={canUpdateCanvases}
              canDeleteCanvases={canDeleteCanvases}
              permissionsLoading={permissionsLoading}
            />
          ))}
        </div>
      ) : (
        <div className="flex min-h-40 flex-col items-center justify-center gap-2 rounded-md px-4 py-8 text-center text-[13px] font-medium text-white/80">
          <FolderOpen size={18} className="text-white/80" />
          <span>No canvases in this group</span>
        </div>
      )}
    </section>
  );
}
