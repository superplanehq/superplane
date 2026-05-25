import type { CanvasesCanvas } from "@/api-client";
import { Command, CommandEmpty, CommandGroup, CommandItem, CommandList } from "@/components/ui/command";
import { usePermissions } from "@/contexts/usePermissions";
import { useCanvases } from "@/hooks/useCanvasData";
import { useRecentCanvasOpens } from "@/hooks/useRecentCanvasOpens";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { cn } from "@/lib/utils";
import { sortCanvasProjectsByRecentOpen, type CanvasProjectOption } from "@/lib/recentCanvasOpens";
import { useCreateApp } from "@/pages/home/useCreateApp";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Command as CommandPrimitive } from "cmdk";
import { Check, Plus, Search } from "lucide-react";
import { useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from "react";
import { useNavigate } from "react-router-dom";

const SWITCHER_HEIGHT_CLASS = "h-7";
const SWITCHER_SEARCH_INPUT_ID = "canvas-project-switcher-search";
const SWITCHER_WIDTH_CLASS = "w-[320px] min-w-[320px] max-w-full";
const SWITCHER_BORDER_CLASS = "border border-slate-950/20";
const SWITCHER_MENU_SURFACE_CLASS = "rounded-md bg-white shadow-md";
const CREATE_APP_VALUE = "__create_app__";

function filterProjects(value: string, search: string, keywords?: string[]) {
  if (value === CREATE_APP_VALUE) {
    return 1;
  }

  const extendedValue = `${value} ${keywords?.join(" ") ?? ""}`.toLowerCase();
  return extendedValue.includes(search.toLowerCase()) ? 1 : 0;
}

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

function useProjectSwitcherKeyboard(open: boolean, setOpen: Dispatch<SetStateAction<boolean>>) {
  const openViaShortcutRef = useRef(false);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (!(event.ctrlKey || event.metaKey) || event.key !== "k") {
        return;
      }

      event.preventDefault();
      openViaShortcutRef.current = true;
      setOpen(true);
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [setOpen]);

  useEffect(() => {
    if (!open || !openViaShortcutRef.current) {
      return;
    }

    openViaShortcutRef.current = false;
    requestAnimationFrame(() => {
      const input = document.getElementById(SWITCHER_SEARCH_INPUT_ID);
      if (input instanceof HTMLInputElement) {
        input.focus();
        input.select();
      }
    });
  }, [open]);
}

interface CanvasProjectSwitcherTriggerProps {
  displayName: string;
  open: boolean;
}

function CanvasProjectSwitcherTrigger({ displayName, open }: CanvasProjectSwitcherTriggerProps) {
  const triggerClassName = cn(
    "flex items-center gap-2 rounded-md bg-white px-2.5 text-[13px] transition-colors",
    SWITCHER_HEIGHT_CLASS,
    "w-full justify-center hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/40",
  );

  return (
    <PopoverTrigger asChild>
      <button
        type="button"
        aria-label="Switch project"
        aria-expanded={open}
        aria-keyshortcuts="Meta+K Control+K"
        title="Search apps (⌘K)"
        data-testid="canvas-project-switcher"
        className={cn(triggerClassName, open && "pointer-events-none invisible")}
      >
        <Search className="size-3.5 shrink-0 text-slate-400" aria-hidden="true" />
        <span className="min-w-0 truncate font-medium text-slate-700">{displayName}</span>
      </button>
    </PopoverTrigger>
  );
}

interface CanvasProjectSwitcherMenuProps {
  isLoading: boolean;
  projects: CanvasProjectOption[];
  activeCanvasId?: string;
  canCreateCanvases: boolean;
  isCreateAppSaving: boolean;
  onSelectProject: (canvasId: string) => void;
  onCreateApp: () => void;
}

function CanvasProjectSwitcherMenu({
  isLoading,
  projects,
  activeCanvasId,
  canCreateCanvases,
  isCreateAppSaving,
  onSelectProject,
  onCreateApp,
}: CanvasProjectSwitcherMenuProps) {
  const searchRowClassName = cn(
    "flex items-center gap-2 bg-white px-2.5 text-[13px]",
    SWITCHER_HEIGHT_CLASS,
    "rounded-none border-0 border-b border-slate-950/10 ring-0",
  );

  return (
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
      <Command shouldFilter filter={filterProjects} className="bg-white">
        <div className={searchRowClassName}>
          <Search className="size-3.5 shrink-0 text-slate-400" aria-hidden="true" />
          <CommandPrimitive.Input
            id={SWITCHER_SEARCH_INPUT_ID}
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
                    onSelect={() => onSelectProject(project.id)}
                    className="cursor-pointer text-[13px]"
                  >
                    <span className="min-w-0 flex-1 truncate">{project.name}</span>
                    {project.id === activeCanvasId ? <Check className="size-3.5 shrink-0 text-slate-600" /> : null}
                  </CommandItem>
                ))}
              </CommandGroup>
            </>
          )}
          {canCreateCanvases ? (
            <CommandGroup className="border-t border-slate-950/10 pt-1 text-[13px]">
              <CommandItem
                value={CREATE_APP_VALUE}
                keywords={["new", "app", "create"]}
                onSelect={onCreateApp}
                disabled={isCreateAppSaving}
                className="cursor-pointer text-[13px]"
              >
                <Plus className="size-3.5 shrink-0 text-slate-500" aria-hidden="true" />
                {isCreateAppSaving ? "Creating..." : "New App"}
              </CommandItem>
            </CommandGroup>
          ) : null}
        </CommandList>
      </Command>
    </PopoverContent>
  );
}

export interface CanvasProjectSwitcherProps {
  organizationId: string;
  activeCanvasId?: string;
  canvasName: string;
}

export function CanvasProjectSwitcher({ organizationId, activeCanvasId, canvasName }: CanvasProjectSwitcherProps) {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const { canAct } = usePermissions();
  const { data: canvases = [], isLoading } = useCanvases(organizationId);
  const { recentOpens, recordOpen } = useRecentCanvasOpens(organizationId);
  const { createApp, isSaving: isCreateAppSaving } = useCreateApp({ onCreated: () => setOpen(false) });

  const canCreateCanvases = canAct("canvases", "create");

  useProjectSwitcherKeyboard(open, setOpen);

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

  const handleSelect = (canvasId: string) => {
    setOpen(false);
    if (canvasId === activeCanvasId) {
      return;
    }

    navigate(`/${organizationId}/canvases/${canvasId}`);
  };

  const handleCreateApp = () => {
    if (!canCreateCanvases || isCreateAppSaving) {
      return;
    }

    setOpen(false);
    void createApp(generateCanvasName());
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <div className={cn("relative mx-auto", SWITCHER_WIDTH_CLASS)}>
        <CanvasProjectSwitcherTrigger displayName={displayName} open={open} />
        <CanvasProjectSwitcherMenu
          isLoading={isLoading}
          projects={projects}
          activeCanvasId={activeCanvasId}
          canCreateCanvases={canCreateCanvases}
          isCreateAppSaving={isCreateAppSaving}
          onSelectProject={handleSelect}
          onCreateApp={handleCreateApp}
        />
      </div>
    </Popover>
  );
}
