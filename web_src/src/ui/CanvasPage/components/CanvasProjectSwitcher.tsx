import type { CanvasesCanvas } from "@/api-client";
import { Input } from "@/components/Input/input";
import { Command, CommandEmpty, CommandGroup, CommandItem, CommandList } from "@/components/ui/command";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useCanvases, useUpdateCanvas } from "@/hooks/useCanvasData";
import { useRecentCanvasOpens } from "@/hooks/useRecentCanvasOpens";
import { getApiErrorMessage } from "@/lib/errors";
import { sortCanvasProjectsByRecentOpen, type CanvasProjectOption } from "@/lib/recentCanvasOpens";
import { showErrorToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Command as CommandPrimitive } from "cmdk";
import { Check, Pencil, Search } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useNavigate } from "react-router-dom";

const SWITCHER_HEIGHT_CLASS = "h-7";
const SWITCHER_WIDTH_CLASS = "w-[320px] min-w-[320px] max-w-full";
const SWITCHER_BORDER_CLASS = "border border-slate-950/20";
const SWITCHER_MENU_SURFACE_CLASS = "rounded-md bg-white shadow-md";

function toCanvasProjectOptions(canvases: CanvasesCanvas[]): CanvasProjectOption[] {
  return canvases
    .map((canvas) => {
      const id = canvas.metadata?.id;
      const name = canvas.metadata?.name;
      if (!id || !name) {
        return null;
      }

      return { id, name };
    })
    .filter((project): project is CanvasProjectOption => project !== null);
}

export interface CanvasProjectSwitcherProps {
  organizationId: string;
  activeCanvasId?: string;
  canvasName: string;
  canUpdateCanvas?: boolean;
}

export function CanvasProjectSwitcher({
  organizationId,
  activeCanvasId,
  canvasName,
  canUpdateCanvas = false,
}: CanvasProjectSwitcherProps) {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [isRenaming, setIsRenaming] = useState(false);
  const [draftName, setDraftName] = useState(canvasName);
  const renameInputRef = useRef<HTMLInputElement>(null);
  const isSubmittingRenameRef = useRef(false);
  const skipRenameBlurSubmitRef = useRef(false);
  const ignoreBlurUntilRef = useRef(0);
  const { data: canvases = [], isLoading } = useCanvases(organizationId);
  const { recentOpens, recordOpen } = useRecentCanvasOpens(organizationId);
  const updateCanvasMutation = useUpdateCanvas(organizationId, activeCanvasId ?? "");

  useEffect(() => {
    if (activeCanvasId) {
      recordOpen(activeCanvasId);
    }
  }, [activeCanvasId, recordOpen]);

  const projects = useMemo(() => {
    const options = toCanvasProjectOptions(canvases);
    return sortCanvasProjectsByRecentOpen(options, recentOpens);
  }, [canvases, recentOpens]);

  const displayName = canvasName.trim() || "Canvas";

  useEffect(() => {
    if (!isRenaming) {
      setDraftName(canvasName);
    }
  }, [canvasName, isRenaming]);

  useEffect(() => {
    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (!(event.ctrlKey || event.metaKey) || event.key !== "k") {
        return;
      }

      const target = event.target;
      if (
        target instanceof Element &&
        target.closest('input, textarea, select, [contenteditable="true"], .monaco-editor')
      ) {
        return;
      }

      if (isRenaming) {
        return;
      }

      event.preventDefault();
      setOpen(true);
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isRenaming]);

  const focusRenameInput = useCallback((selectText = false) => {
    window.setTimeout(() => {
      renameInputRef.current?.focus();
      if (selectText) {
        renameInputRef.current?.select();
      }
    }, 0);
  }, []);

  const cancelRenaming = useCallback(() => {
    skipRenameBlurSubmitRef.current = true;
    setDraftName(canvasName);
    setIsRenaming(false);
  }, [canvasName]);

  const submitRename = useCallback(async () => {
    if (!canUpdateCanvas || !activeCanvasId || isSubmittingRenameRef.current || updateCanvasMutation.isPending) {
      return;
    }

    const name = draftName.trim();
    if (!name) {
      showErrorToast("App name is required");
      focusRenameInput();
      return;
    }

    if (name === displayName) {
      cancelRenaming();
      return;
    }

    isSubmittingRenameRef.current = true;

    try {
      await updateCanvasMutation.mutateAsync({ name });
      setIsRenaming(false);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename app"));
      focusRenameInput();
    } finally {
      isSubmittingRenameRef.current = false;
    }
  }, [activeCanvasId, cancelRenaming, canUpdateCanvas, displayName, draftName, focusRenameInput, updateCanvasMutation]);

  const startRenaming = useCallback(
    ({ preserveFocus = false }: { preserveFocus?: boolean } = {}) => {
      if (!canUpdateCanvas || !activeCanvasId || updateCanvasMutation.isPending) {
        return;
      }

      setOpen(false);

      if (preserveFocus) {
        ignoreBlurUntilRef.current = Date.now() + 200;
      }

      skipRenameBlurSubmitRef.current = false;
      setDraftName(canvasName);
      setIsRenaming(true);
      focusRenameInput(true);
    },
    [activeCanvasId, canUpdateCanvas, canvasName, focusRenameInput, updateCanvasMutation.isPending],
  );

  const handleRenameClick = () => {
    if (isRenaming) {
      focusRenameInput(true);
      return;
    }

    startRenaming();
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

  const handleSelect = (canvasId: string) => {
    setOpen(false);
    if (canvasId === activeCanvasId) {
      return;
    }

    navigate(`/${organizationId}/canvases/${canvasId}`);
  };

  const searchRowClassName = cn(
    "flex items-center gap-2 bg-white px-2.5 text-[13px]",
    SWITCHER_HEIGHT_CLASS,
    "rounded-none border-0 border-b border-slate-950/10 ring-0",
  );

  const triggerSurfaceClassName = cn(
    "rounded-md bg-transparent transition-colors group-hover/switcher:bg-slate-100",
    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/40",
  );

  const triggerIconClassName = "size-3.5 shrink-0 text-slate-400 transition-colors group-hover/trigger:text-slate-800";
  const renameIconClassName = "size-3.5 shrink-0 text-slate-400 transition-colors group-hover/rename:text-slate-800";

  const triggerClassName = cn(
    "group/trigger flex items-center gap-2 px-2.5 text-[13px]",
    SWITCHER_HEIGHT_CLASS,
    "w-full justify-center",
    triggerSurfaceClassName,
  );

  const iconTriggerClassName = cn(
    "group/rename flex items-center justify-center text-[13px]",
    SWITCHER_HEIGHT_CLASS,
    "w-7 shrink-0",
    triggerSurfaceClassName,
  );

  const switcherSurface = isRenaming ? (
    <Input
      ref={renameInputRef}
      value={draftName}
      onChange={(event) => setDraftName(event.target.value)}
      onBlur={() => {
        if (skipRenameBlurSubmitRef.current) {
          skipRenameBlurSubmitRef.current = false;
          return;
        }

        if (ignoreBlurUntilRef.current > Date.now()) {
          focusRenameInput();
          return;
        }

        if (!isSubmittingRenameRef.current) {
          void submitRename();
        }
      }}
      onKeyDown={handleRenameKeyDown}
      aria-label="App name"
      disabled={updateCanvasMutation.isPending}
      data-testid="canvas-rename-input"
      className="h-7 px-2.5 py-0 text-center text-[13px] font-medium"
    />
  ) : (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Switch project"
          aria-expanded={open}
          data-testid="canvas-project-switcher"
          className={cn(triggerClassName, open && "pointer-events-none invisible")}
        >
          <Search className={triggerIconClassName} aria-hidden="true" />
          <span className="min-w-0 truncate font-medium text-slate-700">{displayName}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="center"
        side="bottom"
        sideOffset={-28}
        className={cn(
          SWITCHER_MENU_SURFACE_CLASS,
          SWITCHER_BORDER_CLASS,
          "w-[var(--radix-popover-trigger-width)] overflow-hidden p-0 outline-none",
        )}
      >
        <Command shouldFilter className="bg-white">
          <div className={searchRowClassName}>
            <Search className="size-3.5 shrink-0 text-slate-400" aria-hidden="true" />
            <CommandPrimitive.Input
              placeholder="Search Apps"
              className="min-w-0 flex-1 bg-transparent text-[13px] font-medium text-slate-700 outline-none placeholder:font-normal placeholder:text-slate-400"
            />
          </div>
          <CommandList className="max-h-[280px]">
            {isLoading ? (
              <div className="px-3 py-6 text-center text-[13px] text-slate-500">Loading projects...</div>
            ) : (
              <>
                <CommandEmpty className="py-6 text-[13px]">No projects found.</CommandEmpty>
                <CommandGroup heading="Recently Opened" className="text-[13px]">
                  {projects.map((project) => (
                    <CommandItem
                      key={project.id}
                      value={`${project.name} ${project.id}`}
                      keywords={[project.name]}
                      onSelect={() => handleSelect(project.id)}
                      className="cursor-pointer text-[13px] data-[selected=true]:bg-sky-100 data-[selected=true]:text-slate-900"
                    >
                      <span className="min-w-0 flex-1 truncate">{project.name}</span>
                      {project.id === activeCanvasId ? <Check className="size-3.5 shrink-0 text-slate-600" /> : null}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );

  return (
    <div
      className={cn(
        "group/switcher mx-auto flex items-center gap-1",
        canUpdateCanvas && activeCanvasId && !isRenaming && "-mr-7",
      )}
    >
      <div className={cn("relative", SWITCHER_WIDTH_CLASS)}>{switcherSurface}</div>
      {canUpdateCanvas && activeCanvasId && !isRenaming ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              aria-label="Rename app"
              data-testid="canvas-rename-trigger"
              className={cn(
                iconTriggerClassName,
                "pointer-events-none opacity-0 group-hover/switcher:pointer-events-auto group-hover/switcher:opacity-100 focus-visible:pointer-events-auto focus-visible:opacity-100",
              )}
              disabled={updateCanvasMutation.isPending}
              onClick={handleRenameClick}
            >
              <Pencil className={renameIconClassName} aria-hidden="true" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="right" sideOffset={6}>
            Rename
          </TooltipContent>
        </Tooltip>
      ) : null}
    </div>
  );
}
