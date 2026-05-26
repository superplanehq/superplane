import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";

type CanvasMode = "version-live" | "version-edit" | "runs" | "dashboard" | "memory";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLive: () => void;
  onSelectDashboard?: () => void;
  onSelectMemory?: () => void;
  editing?: boolean;
  hasDraft?: boolean;
}

const CANVAS_TAB = "canvas";
const DASHBOARD_TAB = "dashboard";
const MEMORY_TAB = "memory";
const RUNS_MODE = "runs";

export function CanvasModeToggle({
  mode,
  onSelectLive,
  onSelectDashboard,
  onSelectMemory,
  editing = false,
  hasDraft = false,
}: CanvasModeToggleProps) {
  const showDashboard = Boolean(onSelectDashboard);
  const showMemory = Boolean(onSelectMemory);
  const selected =
    mode === DASHBOARD_TAB
      ? DASHBOARD_TAB
      : mode === MEMORY_TAB
        ? MEMORY_TAB
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
        }
      }}
    >
      <TabsList
        aria-label="Canvas view"
        className={cn(
          "h-7 min-h-7 p-1 [&_[data-slot=tabs-trigger]]:text-[13px]",
          editing
            ? "rounded-full bg-[var(--purple)] text-white [&_[data-slot=tabs-trigger]]:transition-none [&_[data-slot=tabs-trigger][data-state=inactive]]:bg-transparent [&_[data-slot=tabs-trigger][data-state=inactive]]:text-white/90 [&_[data-slot=tabs-trigger][data-state=inactive]]:hover:text-white [&_[data-slot=tabs-trigger][data-state=active]]:rounded-full [&_[data-slot=tabs-trigger][data-state=active]]:bg-white [&_[data-slot=tabs-trigger][data-state=active]]:text-slate-900 [&_[data-slot=tabs-trigger][data-state=active]]:shadow-none"
            : "bg-slate-100 [&_[data-slot=tabs-trigger][data-state=inactive]]:text-slate-500",
        )}
      >
        {showDashboard ? (
          <TabsTrigger value={DASHBOARD_TAB} data-testid="canvas-view-mode-dashboard" aria-label="Console">
            Console
          </TabsTrigger>
        ) : null}
        <TabsTrigger
          value={CANVAS_TAB}
          data-testid="canvas-view-mode-live"
          aria-label={editing ? "Canvas (editing)" : "Canvas"}
        >
          <span className="inline-flex items-center gap-1.5">
            Canvas
            {hasDraft ? (
              <span
                className="inline-flex size-1.5 shrink-0 rounded-full bg-slate-400"
                aria-hidden="true"
                data-testid="canvas-view-mode-live-draft-dot"
              />
            ) : null}
          </span>
        </TabsTrigger>
        {showMemory ? (
          <TabsTrigger value={MEMORY_TAB} data-testid="canvas-view-mode-memory" aria-label="Memory">
            Memory
          </TabsTrigger>
        ) : null}
      </TabsList>
    </Tabs>
  );
}
