import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";
import { DraftChangeDots } from "./DraftChangeDots";

export type CanvasMode = "version-live" | "version-edit" | "runs" | "dashboard" | "memory" | "files";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLive: () => void;
  onSelectDashboard?: () => void;
  onSelectMemory?: () => void;
  onSelectFiles?: () => void;
  editing?: boolean;
  /** @deprecated Use hasCanvasUncommitted / hasCanvasCommitted */
  hasDraft?: boolean;
  /** @deprecated Use hasDashboardUncommitted / hasDashboardCommitted */
  hasDashboardDraft?: boolean;
  hasCanvasUncommitted?: boolean;
  hasCanvasCommitted?: boolean;
  hasDashboardUncommitted?: boolean;
  hasDashboardCommitted?: boolean;
  hasFilesUncommitted?: boolean;
  /** Edit-mode tab bar color aligned with draft status badges. */
  editTabTone?: "uncommitted" | "ready" | "neutral";
}

const EDITING_TAB_TRIGGER_ACTIVE =
  "[&_[data-slot=tabs-trigger][data-state=active]]:rounded-full [&_[data-slot=tabs-trigger][data-state=active]]:bg-white [&_[data-slot=tabs-trigger][data-state=active]]:text-slate-900 [&_[data-slot=tabs-trigger][data-state=active]]:shadow-sm";
const EDITING_TAB_TRIGGER_BASE =
  "[&_[data-slot=tabs-trigger]]:transition-none [&_[data-slot=tabs-trigger][data-state=inactive]]:bg-transparent";

function editingTabListClassName(tone: CanvasModeToggleProps["editTabTone"]): string {
  if (tone === "uncommitted") {
    return cn(
      "rounded-full bg-orange-50",
      EDITING_TAB_TRIGGER_BASE,
      "[&_[data-slot=tabs-trigger][data-state=inactive]]:text-orange-800/80 [&_[data-slot=tabs-trigger][data-state=inactive]]:hover:text-orange-900",
      EDITING_TAB_TRIGGER_ACTIVE,
    );
  }

  if (tone === "ready") {
    return cn(
      "rounded-full bg-blue-50",
      EDITING_TAB_TRIGGER_BASE,
      "[&_[data-slot=tabs-trigger][data-state=inactive]]:text-blue-800/80 [&_[data-slot=tabs-trigger][data-state=inactive]]:hover:text-blue-900",
      EDITING_TAB_TRIGGER_ACTIVE,
    );
  }

  return cn(
    "rounded-full bg-slate-100",
    EDITING_TAB_TRIGGER_BASE,
    "[&_[data-slot=tabs-trigger][data-state=inactive]]:text-slate-600 [&_[data-slot=tabs-trigger][data-state=inactive]]:hover:text-slate-900",
    EDITING_TAB_TRIGGER_ACTIVE,
  );
}

const CANVAS_TAB = "canvas";
const DASHBOARD_TAB = "dashboard";
const MEMORY_TAB = "memory";
const FILES_TAB = "files";
const RUNS_MODE = "runs";

export function CanvasModeToggle({
  mode,
  onSelectLive,
  onSelectDashboard,
  onSelectMemory,
  onSelectFiles,
  editing = false,
  hasDraft = false,
  hasDashboardDraft = false,
  hasCanvasUncommitted,
  hasCanvasCommitted,
  hasDashboardUncommitted,
  hasDashboardCommitted,
  hasFilesUncommitted,
  editTabTone = "neutral",
}: CanvasModeToggleProps) {
  const canvasUncommitted = hasCanvasUncommitted ?? hasDraft;
  const canvasCommitted = hasCanvasCommitted ?? false;
  const dashboardUncommitted = hasDashboardUncommitted ?? hasDashboardDraft;
  const dashboardCommitted = hasDashboardCommitted ?? false;
  const filesUncommitted = hasFilesUncommitted ?? false;
  const showDashboard = Boolean(onSelectDashboard);
  const showMemory = Boolean(onSelectMemory);
  const showFiles = Boolean(onSelectFiles);
  const selected =
    mode === DASHBOARD_TAB
      ? DASHBOARD_TAB
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

        if (next === DASHBOARD_TAB && selected !== DASHBOARD_TAB && onSelectDashboard) {
          valueChangeHandledRef.current = true;
          queueMicrotask(() => {
            valueChangeHandledRef.current = false;
          });
          void onSelectDashboard();
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
            ? editingTabListClassName(editTabTone)
            : "rounded-full bg-slate-100 [&_[data-slot=tabs-trigger][data-state=inactive]]:text-slate-500 [&_[data-slot=tabs-trigger][data-state=active]]:rounded-full",
        )}
      >
        {showDashboard ? (
          <TabsTrigger value={DASHBOARD_TAB} data-testid="canvas-view-mode-dashboard" aria-label="Console">
            <span className="inline-flex items-center gap-1.5">
              Console
              <DraftChangeDots
                uncommitted={dashboardUncommitted}
                committed={dashboardCommitted}
                testIdPrefix="canvas-view-mode-dashboard"
              />
            </span>
          </TabsTrigger>
        ) : null}
        <TabsTrigger
          value={CANVAS_TAB}
          data-testid="canvas-view-mode-live"
          aria-label={editing ? "Canvas (editing)" : "Canvas"}
        >
          <span className="inline-flex items-center gap-1.5">
            Canvas
            <DraftChangeDots
              uncommitted={canvasUncommitted}
              committed={canvasCommitted}
              testIdPrefix="canvas-view-mode-live"
            />
          </span>
        </TabsTrigger>
        {showMemory ? (
          <TabsTrigger value={MEMORY_TAB} data-testid="canvas-view-mode-memory" aria-label="Memory">
            Memory
          </TabsTrigger>
        ) : null}
        {showFiles ? (
          <TabsTrigger value={FILES_TAB} data-testid="canvas-view-mode-files" aria-label="Files">
            <span className="inline-flex items-center gap-1.5">
              Files
              <DraftChangeDots uncommitted={filesUncommitted} committed={false} testIdPrefix="canvas-view-mode-files" />
            </span>
          </TabsTrigger>
        ) : null}
      </TabsList>
    </Tabs>
  );
}
