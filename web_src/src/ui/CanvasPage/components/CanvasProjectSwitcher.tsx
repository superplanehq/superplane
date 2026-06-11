import type { CanvasesCanvasSummary } from "@/api-client";
import { Input } from "@/components/Input/input";
import { Command, CommandEmpty, CommandGroup, CommandItem, CommandList } from "@/components/ui/command";
import { useCanvases, useUpdateCanvas } from "@/hooks/useCanvasData";
import { isNormalClick } from "@/lib/linkHelpers";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useRecentCanvasOpens } from "@/hooks/useRecentCanvasOpens";
import { getApiErrorMessage } from "@/lib/errors";
import { sortCanvasProjectsByRecentOpen, type CanvasProjectOption } from "@/lib/recentCanvasOpens";
import { appPath, appSettingsPath } from "@/lib/appPaths";
import { showErrorToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Command as CommandPrimitive } from "cmdk";
import { Check, MoreVertical, Pencil, Search, Settings } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent, type RefObject } from "react";
import { Link, useNavigate } from "react-router-dom";

const SWITCHER_HEIGHT_CLASS = "h-7";
const SWITCHER_WIDTH_CLASS = "w-[320px] min-w-[320px] max-w-full";
const SWITCHER_BORDER_CLASS = "border border-slate-950/20";
const SWITCHER_MENU_SURFACE_CLASS = "rounded-md bg-white shadow-md";
const SEARCH_ROW_CLASS_NAME = cn(
  "flex items-center gap-2 bg-white px-2.5 text-[13px]",
  SWITCHER_HEIGHT_CLASS,
  "rounded-none border-0 border-b border-slate-950/10 ring-0",
);
const TRIGGER_SURFACE_CLASS_NAME = cn(
  "rounded-md bg-transparent transition-colors group-hover/switcher:bg-slate-100",
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/40",
);
const TRIGGER_ICON_CLASS_NAME = "size-3.5 shrink-0 text-slate-400 transition-colors group-hover/trigger:text-slate-800";
const ACTIONS_ICON_CLASS_NAME = "size-3.5 shrink-0 text-slate-400 transition-colors group-hover/actions:text-slate-800";
const TRIGGER_CLASS_NAME = cn(
  "group/trigger flex items-center gap-2 px-2.5 text-[13px]",
  SWITCHER_HEIGHT_CLASS,
  "w-full justify-center",
  TRIGGER_SURFACE_CLASS_NAME,
);
const ACTIONS_TRIGGER_CLASS_NAME = cn(
  "group/actions flex items-center justify-center text-[13px]",
  SWITCHER_HEIGHT_CLASS,
  "w-7 shrink-0",
  TRIGGER_SURFACE_CLASS_NAME,
);

function toCanvasProjectOptions(canvases: CanvasesCanvasSummary[]): CanvasProjectOption[] {
  return canvases
    .map((canvas) => {
      const id = canvas.id;
      const name = canvas.name;
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
  const { data: canvases = [], isLoading } = useCanvases(organizationId);
  const { recentOpens, recordOpen } = useRecentCanvasOpens(organizationId);

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
  const rename = useCanvasProjectRename({
    organizationId,
    activeCanvasId,
    canvasName,
    canUpdateCanvas,
    setProjectSearchOpen: setOpen,
  });
  useCanvasProjectShortcut(rename.isRenaming, setOpen);

  const handleOpenSettings = useCallback(() => {
    if (!activeCanvasId) {
      return;
    }

    navigate(appSettingsPath(organizationId, activeCanvasId));
  }, [activeCanvasId, navigate, organizationId]);

  const handleRenameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      void rename.submitRename();
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      rename.cancelRenaming();
    }
  };

  const handleSelect = (canvasId: string) => {
    setOpen(false);
    if (canvasId === activeCanvasId) {
      return;
    }

    navigate(appPath(organizationId, canvasId));
  };

  const switcherSurface = rename.isRenaming ? (
    <CanvasRenameInput
      inputRef={rename.inputRef}
      draftName={rename.draftName}
      onDraftNameChange={rename.setDraftName}
      skipBlurSubmitRef={rename.skipBlurSubmitRef}
      isSubmittingRef={rename.isSubmittingRef}
      submitRename={rename.submitRename}
      onKeyDown={handleRenameKeyDown}
      isPending={rename.isPending}
    />
  ) : (
    <ProjectSearchPopover
      open={open}
      onOpenChange={setOpen}
      displayName={displayName}
      isLoading={isLoading}
      projects={projects}
      activeCanvasId={activeCanvasId}
      onSelect={handleSelect}
    />
  );

  return (
    <div
      className={cn(
        "group/switcher mx-auto flex items-center gap-1",
        canUpdateCanvas && activeCanvasId && !rename.isRenaming && "-mr-7",
      )}
    >
      <div className={cn("relative", SWITCHER_WIDTH_CLASS)}>{switcherSurface}</div>
      {canUpdateCanvas && activeCanvasId && !rename.isRenaming ? (
        <ProjectActionsMenu
          isPending={rename.isPending}
          onRename={rename.startRenaming}
          onOpenSettings={handleOpenSettings}
        />
      ) : null}
    </div>
  );
}

function useCanvasProjectShortcut(isRenaming: boolean, setOpen: (open: boolean) => void) {
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
  }, [isRenaming, setOpen]);
}

function useCanvasProjectRename({
  organizationId,
  activeCanvasId,
  canvasName,
  canUpdateCanvas,
  setProjectSearchOpen,
}: {
  organizationId: string;
  activeCanvasId?: string;
  canvasName: string;
  canUpdateCanvas: boolean;
  setProjectSearchOpen: (open: boolean) => void;
}) {
  const [isRenaming, setIsRenaming] = useState(false);
  const [draftName, setDraftName] = useState(canvasName);
  const inputRef = useRef<HTMLInputElement>(null);
  const isSubmittingRef = useRef(false);
  const skipBlurSubmitRef = useRef(false);
  const updateCanvasMutation = useUpdateCanvas(organizationId, activeCanvasId ?? "");

  useEffect(() => {
    if (!isRenaming) {
      setDraftName(canvasName);
    }
  }, [canvasName, isRenaming]);

  const focusInput = useCallback((selectText = false) => {
    window.setTimeout(() => {
      inputRef.current?.focus();
      if (selectText) {
        inputRef.current?.select();
      }
    }, 0);
  }, []);

  const cancelRenaming = useCallback(() => {
    skipBlurSubmitRef.current = true;
    setDraftName(canvasName);
    setIsRenaming(false);
  }, [canvasName]);

  const submitRename = useCallback(async () => {
    if (!canUpdateCanvas || !activeCanvasId || isSubmittingRef.current || updateCanvasMutation.isPending) {
      return;
    }

    const name = draftName.trim();
    if (!name) {
      showErrorToast("App name is required");
      focusInput();
      return;
    }

    if (name === canvasName.trim()) {
      cancelRenaming();
      return;
    }

    isSubmittingRef.current = true;

    try {
      await updateCanvasMutation.mutateAsync({ name });
      skipBlurSubmitRef.current = true;
      setIsRenaming(false);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename app"));
      focusInput();
    } finally {
      isSubmittingRef.current = false;
    }
  }, [activeCanvasId, cancelRenaming, canUpdateCanvas, canvasName, draftName, focusInput, updateCanvasMutation]);

  const startRenaming = useCallback(() => {
    if (!canUpdateCanvas || !activeCanvasId || updateCanvasMutation.isPending) {
      return;
    }

    setProjectSearchOpen(false);
    skipBlurSubmitRef.current = false;
    setDraftName(canvasName);
    setIsRenaming(true);
    focusInput(true);
  }, [activeCanvasId, canUpdateCanvas, canvasName, focusInput, setProjectSearchOpen, updateCanvasMutation.isPending]);

  return {
    isRenaming,
    draftName,
    setDraftName,
    inputRef,
    isSubmittingRef,
    skipBlurSubmitRef,
    focusInput,
    cancelRenaming,
    submitRename,
    startRenaming,
    isPending: updateCanvasMutation.isPending,
  };
}

function CanvasRenameInput({
  inputRef,
  draftName,
  onDraftNameChange,
  skipBlurSubmitRef,
  isSubmittingRef,
  submitRename,
  onKeyDown,
  isPending,
}: {
  inputRef: RefObject<HTMLInputElement | null>;
  draftName: string;
  onDraftNameChange: (value: string) => void;
  skipBlurSubmitRef: RefObject<boolean>;
  isSubmittingRef: RefObject<boolean>;
  submitRename: () => Promise<void>;
  onKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
  isPending: boolean;
}) {
  const handleBlur = () => {
    if (skipBlurSubmitRef.current) {
      skipBlurSubmitRef.current = false;
      return;
    }

    if (!isSubmittingRef.current) {
      void submitRename();
    }
  };

  return (
    <Input
      ref={inputRef}
      value={draftName}
      onChange={(event) => onDraftNameChange(event.target.value)}
      onBlur={handleBlur}
      onKeyDown={onKeyDown}
      aria-label="App name"
      disabled={isPending}
      data-testid="canvas-rename-input"
      className="h-7 px-2.5 py-0 text-center text-[13px] font-medium"
    />
  );
}

function ProjectSearchPopover({
  open,
  onOpenChange,
  displayName,
  isLoading,
  projects,
  activeCanvasId,
  onSelect,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  displayName: string;
  isLoading: boolean;
  projects: CanvasProjectOption[];
  activeCanvasId?: string;
  onSelect: (canvasId: string) => void;
}) {
  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Switch project"
          aria-expanded={open}
          data-testid="canvas-project-switcher"
          className={cn(TRIGGER_CLASS_NAME, open && "pointer-events-none invisible")}
        >
          <Search className={TRIGGER_ICON_CLASS_NAME} aria-hidden="true" />
          <span className="min-w-0 truncate font-medium text-slate-800">{displayName}</span>
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
          <div className={SEARCH_ROW_CLASS_NAME}>
            <Search className="size-3.5 shrink-0 text-slate-400" aria-hidden="true" />
            <CommandPrimitive.Input
              placeholder="Search Apps"
              className="min-w-0 flex-1 bg-transparent text-[13px] font-medium text-slate-700 outline-none placeholder:font-normal placeholder:text-slate-400"
            />
          </div>
          <ProjectSearchList
            isLoading={isLoading}
            projects={projects}
            activeCanvasId={activeCanvasId}
            onSelect={onSelect}
          />
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function ProjectSearchList({
  isLoading,
  projects,
  activeCanvasId,
  onSelect,
}: {
  isLoading: boolean;
  projects: CanvasProjectOption[];
  activeCanvasId?: string;
  onSelect: (canvasId: string) => void;
}) {
  const organizationId = useOrganizationId() ?? "";
  if (isLoading) {
    return (
      <CommandList className="max-h-[280px]">
        <div className="px-3 py-6 text-center text-[13px] text-slate-500">Loading projects...</div>
      </CommandList>
    );
  }

  return (
    <CommandList className="max-h-[280px]">
      <CommandEmpty className="py-6 text-[13px]">No projects found.</CommandEmpty>
      <CommandGroup heading="Recently Opened" className="text-[13px]">
        {projects.map((project) => (
          <CommandItem
            key={project.id}
            value={`${project.name} ${project.id}`}
            keywords={[project.name]}
            onSelect={() => onSelect(project.id)}
            className="cursor-pointer text-[13px] data-[selected=true]:bg-sky-100 data-[selected=true]:text-slate-900"
            asChild
          >
            <Link
              to={appPath(organizationId, project.id)}
              onClick={(e) => {
                if (isNormalClick(e)) e.preventDefault();
              }}
            >
              <span className="min-w-0 flex-1 truncate">{project.name}</span>
              {project.id === activeCanvasId ? <Check className="size-3.5 shrink-0 text-slate-600" /> : null}
            </Link>
          </CommandItem>
        ))}
      </CommandGroup>
    </CommandList>
  );
}

function ProjectActionsMenu({
  isPending,
  onRename,
  onOpenSettings,
}: {
  isPending: boolean;
  onRename: () => void;
  onOpenSettings: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label="App actions"
          data-testid="canvas-actions-trigger"
          className={cn(
            ACTIONS_TRIGGER_CLASS_NAME,
            "pointer-events-none opacity-0 group-hover/switcher:pointer-events-auto group-hover/switcher:opacity-100 focus-visible:pointer-events-auto focus-visible:opacity-100",
          )}
          disabled={isPending}
        >
          <MoreVertical className={ACTIONS_ICON_CLASS_NAME} aria-hidden="true" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onRename}>
          <Pencil size={16} />
          Rename
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onOpenSettings}>
          <Settings size={16} />
          App Settings
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
