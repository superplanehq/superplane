import { appPath } from "@/lib/appPaths";
import { isNormalClick } from "@/lib/linkHelpers";
import { segmentedNavTabClassName } from "@/lib/segmentedNav";
import { cn } from "@/lib/utils";
import { Link, useParams } from "react-router-dom";

import { DraftChangeDots } from "./DraftChangeDots";

export type CanvasMode = "version-live" | "console" | "memory" | "files";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLive: () => void;
  onSelectConsole?: () => void;
  onSelectMemory?: () => void;
  onSelectFiles?: () => void;
  editing?: boolean;
  hasCanvasUncommitted?: boolean;
  hasCanvasCommitted?: boolean;
  hasConsoleUncommitted?: boolean;
  hasConsoleCommitted?: boolean;
  hasFilesUncommitted?: boolean;
  hasFilesCommitted?: boolean;
}

const CANVAS_TAB = "canvas";
const CONSOLE_TAB = "console";
const MEMORY_TAB = "memory";
const FILES_TAB = "files";

const MODE_TO_TAB: Record<string, string> = {
  console: CONSOLE_TAB,
  memory: MEMORY_TAB,
  files: FILES_TAB,
};

/** On normal clicks, prevent Link navigation and use the callback (which preserves query params via setSearchParams). */
function handleTabClick(e: React.MouseEvent, isActive: boolean, callback: () => void) {
  if (isNormalClick(e)) {
    e.preventDefault();
    if (!isActive) callback();
  }
}

function modeToTab(mode: string): string {
  return MODE_TO_TAB[mode] ?? CANVAS_TAB;
}

/** Edit-mode nav background tinted to match edit-session chrome. */
function editingNavClassName(): string {
  return "bg-orange-200";
}

/** Edit-mode inactive tab text on the orange nav track. */
function editingInactiveClassName(): string {
  return "bg-transparent text-orange-950/80 hover:text-orange-950 transition-none";
}

/** Active edit tab — white pill on the orange nav track in light mode. */
function editingActiveClassName(): string {
  return "rounded-full bg-white text-gray-800 shadow-sm dark:bg-gray-800 dark:text-gray-100 dark:shadow-none";
}

function tabClasses(selected: string, value: string, editing: boolean) {
  const isActive = selected === value;
  if (!editing) {
    return segmentedNavTabClassName(isActive);
  }

  return segmentedNavTabClassName(isActive, {
    activeClasses: editingActiveClassName(),
    inactiveClasses: editingInactiveClassName(),
  });
}

export function CanvasModeToggle({
  mode,
  onSelectLive,
  onSelectConsole,
  onSelectMemory,
  onSelectFiles,
  editing = false,
  hasCanvasUncommitted = false,
  hasCanvasCommitted = false,
  hasConsoleUncommitted = false,
  hasConsoleCommitted = false,
  hasFilesUncommitted = false,
  hasFilesCommitted = false,
}: CanvasModeToggleProps) {
  const { organizationId, appId } = useParams<{ organizationId: string; appId: string }>();
  const showConsole = Boolean(onSelectConsole);
  const showMemory = Boolean(onSelectMemory);
  const showFiles = Boolean(onSelectFiles);
  const selected = modeToTab(mode);
  const baseHref = organizationId && appId ? appPath(organizationId, appId) : "#";
  const tabHref = (view?: string) => (view ? `${baseHref}?view=${view}` : baseHref);

  return (
    <nav
      aria-label="Canvas view"
      className={cn(
        "inline-flex h-7 min-h-7 items-center justify-center gap-0 rounded-full p-1",
        editing ? editingNavClassName() : "bg-slate-100 dark:bg-gray-800",
      )}
    >
      <Link
        to={tabHref()}
        onClick={(e) => handleTabClick(e, selected === CANVAS_TAB, () => void onSelectLive())}
        className={tabClasses(selected, CANVAS_TAB, editing)}
        data-testid="canvas-view-mode-live"
        aria-label={editing ? "Canvas (editing)" : "Canvas"}
        aria-current={selected === CANVAS_TAB ? "page" : undefined}
      >
        <span className="inline-flex items-center gap-1.5">
          Canvas
          <DraftChangeDots
            uncommitted={hasCanvasUncommitted}
            committed={hasCanvasCommitted}
            testIdPrefix="canvas-view-mode-live"
          />
        </span>
      </Link>
      {showConsole ? (
        <Link
          to={tabHref("console")}
          onClick={(e) => handleTabClick(e, selected === CONSOLE_TAB, () => void onSelectConsole?.())}
          className={tabClasses(selected, CONSOLE_TAB, editing)}
          data-testid="canvas-view-mode-console"
          aria-label="Console"
          aria-current={selected === CONSOLE_TAB ? "page" : undefined}
        >
          <span className="inline-flex items-center gap-1.5">
            Console
            <DraftChangeDots
              uncommitted={hasConsoleUncommitted}
              committed={hasConsoleCommitted}
              testIdPrefix="canvas-view-mode-console"
            />
          </span>
        </Link>
      ) : null}
      {showMemory ? (
        <Link
          to={tabHref("memory")}
          onClick={(e) => handleTabClick(e, selected === MEMORY_TAB, () => void onSelectMemory?.())}
          className={tabClasses(selected, MEMORY_TAB, editing)}
          data-testid="canvas-view-mode-memory"
          aria-label="Memory"
          aria-current={selected === MEMORY_TAB ? "page" : undefined}
        >
          Memory
        </Link>
      ) : null}
      {showFiles ? (
        <Link
          to={tabHref("files")}
          onClick={(e) => handleTabClick(e, selected === FILES_TAB, () => void onSelectFiles?.())}
          className={tabClasses(selected, FILES_TAB, editing)}
          data-testid="canvas-view-mode-files"
          aria-label="Files"
          aria-current={selected === FILES_TAB ? "page" : undefined}
        >
          <span className="inline-flex items-center gap-1.5">
            Files
            <DraftChangeDots
              uncommitted={hasFilesUncommitted}
              committed={hasFilesCommitted}
              testIdPrefix="canvas-view-mode-files"
            />
          </span>
        </Link>
      ) : null}
    </nav>
  );
}
