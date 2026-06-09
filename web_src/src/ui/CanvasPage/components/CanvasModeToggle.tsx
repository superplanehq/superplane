import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { Link, useParams } from "react-router-dom";

export type CanvasMode = "version-live" | "version-edit" | "runs" | "console" | "memory" | "files";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLive: () => void;
  onSelectConsole?: () => void;
  onSelectMemory?: () => void;
  onSelectFiles?: () => void;
  editing?: boolean;
  hasDraft?: boolean;
  hasConsoleDraft?: boolean;
}

const CANVAS_TAB = "canvas";
const CONSOLE_TAB = "console";
const MEMORY_TAB = "memory";
const FILES_TAB = "files";
const RUNS_MODE = "runs";

const BASE_TAB_CLASSES =
  "inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-full border border-transparent px-2.5 py-1 text-[13px] font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] focus-visible:outline-1 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-50";

const ACTIVE_CLASSES = "bg-background text-foreground shadow-sm";
const INACTIVE_CLASSES = "text-slate-500 hover:text-foreground";
const EDITING_ACTIVE_CLASSES = "rounded-full bg-white text-slate-900 shadow-sm";
const EDITING_INACTIVE_CLASSES = "bg-transparent text-blue-800/80 hover:text-blue-900 transition-none";

const MODE_TO_TAB: Record<string, string> = {
  console: CONSOLE_TAB,
  memory: MEMORY_TAB,
  files: FILES_TAB,
  runs: RUNS_MODE,
};

/** On normal clicks, prevent Link navigation and use the callback (which preserves query params via setSearchParams). */
function handleTabClick(e: React.MouseEvent, callback: () => void) {
  if (!e.metaKey && !e.ctrlKey && !e.shiftKey && !e.altKey && e.button === 0) {
    e.preventDefault();
    callback();
  }
}

function modeToTab(mode: string): string {
  return MODE_TO_TAB[mode] ?? CANVAS_TAB;
}

function tabClasses(selected: string, value: string, editing: boolean) {
  const isActive = selected === value;
  const stateClass = isActive
    ? editing
      ? EDITING_ACTIVE_CLASSES
      : ACTIVE_CLASSES
    : editing
      ? EDITING_INACTIVE_CLASSES
      : INACTIVE_CLASSES;
  return cn(BASE_TAB_CLASSES, stateClass);
}

export function CanvasModeToggle({
  mode,
  onSelectLive,
  onSelectConsole,
  onSelectMemory,
  onSelectFiles,
  editing = false,
  hasDraft = false,
  hasConsoleDraft = false,
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
        editing ? "bg-blue-50" : "bg-slate-100",
      )}
    >
      {showConsole ? (
        <Link
          to={tabHref("console")}
          onClick={(e) => handleTabClick(e, () => void onSelectConsole?.())}
          className={tabClasses(selected, CONSOLE_TAB, editing)}
          data-testid="canvas-view-mode-console"
          aria-label="Console"
          aria-current={selected === CONSOLE_TAB ? "page" : undefined}
        >
          <span className="inline-flex items-center gap-1.5">
            Console
            <DraftDot show={hasConsoleDraft} editing={editing} testId="canvas-view-mode-console-draft-dot" />
          </span>
        </Link>
      ) : null}
      <Link
        to={tabHref()}
        onClick={(e) => handleTabClick(e, () => void onSelectLive())}
        className={tabClasses(selected, CANVAS_TAB, editing)}
        data-testid="canvas-view-mode-live"
        aria-label={editing ? "Canvas (editing)" : "Canvas"}
        aria-current={selected === CANVAS_TAB ? "page" : undefined}
      >
        <span className="inline-flex items-center gap-1.5">
          Canvas
          <DraftDot show={hasDraft} editing={editing} testId="canvas-view-mode-live-draft-dot" />
        </span>
      </Link>
      {showMemory ? (
        <Link
          to={tabHref("memory")}
          onClick={(e) => handleTabClick(e, () => void onSelectMemory?.())}
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
          onClick={(e) => handleTabClick(e, () => void onSelectFiles?.())}
          className={tabClasses(selected, FILES_TAB, editing)}
          data-testid="canvas-view-mode-files"
          aria-label="Files"
          aria-current={selected === FILES_TAB ? "page" : undefined}
        >
          Files
        </Link>
      ) : null}
    </nav>
  );
}

function DraftDot({ show, editing, testId }: { show: boolean; editing: boolean; testId: string }) {
  if (!show) {
    return null;
  }

  return (
    <span
      className={cn("inline-flex size-1.5 shrink-0 rounded-full", editing ? "bg-blue-500" : "bg-slate-400")}
      aria-hidden="true"
      data-testid={testId}
    />
  );
}
