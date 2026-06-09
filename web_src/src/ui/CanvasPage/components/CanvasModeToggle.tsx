import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";
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

const EDITING_TAB_TRIGGER_ACTIVE =
  "[&_[data-slot=tabs-trigger][data-state=active]]:rounded-full [&_[data-slot=tabs-trigger][data-state=active]]:bg-white [&_[data-slot=tabs-trigger][data-state=active]]:text-slate-900 [&_[data-slot=tabs-trigger][data-state=active]]:shadow-sm";
const EDITING_TAB_TRIGGER_BASE =
  "[&_[data-slot=tabs-trigger]]:transition-none [&_[data-slot=tabs-trigger][data-state=inactive]]:bg-transparent";

function editingTabListClassName(): string {
  return cn(
    "rounded-full bg-blue-50",
    EDITING_TAB_TRIGGER_BASE,
    "[&_[data-slot=tabs-trigger][data-state=inactive]]:text-blue-800/80 [&_[data-slot=tabs-trigger][data-state=inactive]]:hover:text-blue-900",
    EDITING_TAB_TRIGGER_ACTIVE,
  );
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
  const selected =
    mode === CONSOLE_TAB
      ? CONSOLE_TAB
      : mode === MEMORY_TAB
        ? MEMORY_TAB
        : mode === FILES_TAB
          ? FILES_TAB
          : mode === RUNS_MODE
            ? RUNS_MODE
            : CANVAS_TAB;
  const valueChangeHandledRef = useRef(false);

  useEffect(() => {
    valueChangeHandledRef.current = false;
  }, [mode]);

  return (
    <Tabs
      value={selected}
      onValueChange={(next) => {
        // Radix Tabs may emit more than one `onValueChange` per click when `value` is controlled and the parent
        // doesn't update it synchronously. We only want to suppress duplicates from the same user interaction,
        // not block subsequent clicks.
        if (valueChangeHandledRef.current) return;

        if (next === CANVAS_TAB && selected !== CANVAS_TAB) {
          valueChangeHandledRef.current = true;
          queueMicrotask(() => {
            valueChangeHandledRef.current = false;
          });
          void onSelectLive();
          return;
        }

        if (next === CONSOLE_TAB && selected !== CONSOLE_TAB && onSelectConsole) {
          valueChangeHandledRef.current = true;
          queueMicrotask(() => {
            valueChangeHandledRef.current = false;
          });
          void onSelectConsole();
          return;
        }

        if (next === MEMORY_TAB && selected !== MEMORY_TAB && onSelectMemory) {
          valueChangeHandledRef.current = true;
          queueMicrotask(() => {
            valueChangeHandledRef.current = false;
          });
          void onSelectMemory();
          return;
        }

        if (next === FILES_TAB && selected !== FILES_TAB && onSelectFiles) {
          valueChangeHandledRef.current = true;
          queueMicrotask(() => {
            valueChangeHandledRef.current = false;
          });
          void onSelectFiles();
        }
      }}
    >
      <TabsList
        aria-label="Canvas view"
        className={cn(
          "h-7 min-h-7 p-1 [&_[data-slot=tabs-trigger]]:text-[13px]",
          editing
            ? editingTabListClassName()
            : "rounded-full bg-slate-100 [&_[data-slot=tabs-trigger][data-state=inactive]]:text-slate-500 [&_[data-slot=tabs-trigger][data-state=active]]:rounded-full",
        )}
      >
        {showConsole ? (
          <TabsTrigger value={CONSOLE_TAB} data-testid="canvas-view-mode-console" aria-label="Console" asChild>
            <Link
              to={organizationId && appId ? appPath(organizationId, appId, "?view=console") : "#"}
              onClick={(e) => { if (!(e.metaKey || e.ctrlKey || e.shiftKey || e.altKey)) e.preventDefault(); }}
            >
              <span className="inline-flex items-center gap-1.5">
                Console
                <DraftDot show={hasConsoleDraft} editing={editing} testId="canvas-view-mode-console-draft-dot" />
              </span>
            </Link>
          </TabsTrigger>
        ) : null}
        <TabsTrigger
          value={CANVAS_TAB}
          data-testid="canvas-view-mode-live"
          aria-label={editing ? "Canvas (editing)" : "Canvas"}
          asChild
        >
          <Link
            to={organizationId && appId ? appPath(organizationId, appId) : "#"}
            onClick={(e) => { if (!(e.metaKey || e.ctrlKey || e.shiftKey || e.altKey)) e.preventDefault(); }}
          >
            <span className="inline-flex items-center gap-1.5">
              Canvas
              <DraftDot show={hasDraft} editing={editing} testId="canvas-view-mode-live-draft-dot" />
            </span>
          </Link>
        </TabsTrigger>
        {showMemory ? (
          <TabsTrigger value={MEMORY_TAB} data-testid="canvas-view-mode-memory" aria-label="Memory" asChild>
            <Link
              to={organizationId && appId ? appPath(organizationId, appId, "?view=memory") : "#"}
              onClick={(e) => { if (!(e.metaKey || e.ctrlKey || e.shiftKey || e.altKey)) e.preventDefault(); }}
            >
              Memory
            </Link>
          </TabsTrigger>
        ) : null}
        {showFiles ? (
          <TabsTrigger value={FILES_TAB} data-testid="canvas-view-mode-files" aria-label="Files" asChild>
            <Link
              to={organizationId && appId ? appPath(organizationId, appId, "?view=files") : "#"}
              onClick={(e) => { if (!(e.metaKey || e.ctrlKey || e.shiftKey || e.altKey)) e.preventDefault(); }}
            >
              Files
            </Link>
          </TabsTrigger>
        ) : null}
      </TabsList>
    </Tabs>
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
