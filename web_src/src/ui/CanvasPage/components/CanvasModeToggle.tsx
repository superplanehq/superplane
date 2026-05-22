import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useEffect, useRef } from "react";

type CanvasMode = "version-live" | "version-edit" | "runs" | "dashboard" | "memory";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLive: () => void;
  onSelectDashboard?: () => void;
  onSelectMemory?: () => void;
  /** Distinct namespace count shown as a badge next to the Memory tab. No badge when 0. */
  memoryNamespaceCount?: number;
  editing?: boolean;
  hasDraft?: boolean;
}

const CANVAS_TAB = "canvas";
const DASHBOARD_TAB = "dashboard";
const MEMORY_TAB = "memory";
const RUNS_MODE = "runs";

/** Re-enable when the Memory tab namespace count badge should be visible again. */
const MEMORY_TAB_NAMESPACE_BADGE_ENABLED = false;

export function CanvasModeToggle({
  mode,
  onSelectLive,
  onSelectDashboard,
  onSelectMemory,
  memoryNamespaceCount = 0,
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
      <TabsList aria-label="Canvas view" className="h-8 min-h-8 bg-slate-100 [&_[data-slot=tabs-trigger]]:text-[13px]">
        {showDashboard ? (
          <TabsTrigger value={DASHBOARD_TAB} data-testid="canvas-view-mode-dashboard" aria-label="Dashboard">
            Dashboard
          </TabsTrigger>
        ) : null}
        <TabsTrigger
          value={CANVAS_TAB}
          data-testid="canvas-view-mode-live"
          aria-label={editing ? "Canvas (editing)" : hasDraft ? "Canvas (unpublished draft)" : "Canvas"}
        >
          <span className="inline-flex items-center gap-1.5">
            Canvas
            {hasDraft ? (
              <span
                className="inline-flex size-1.5 shrink-0 rounded-full bg-muted-foreground/70"
                aria-hidden="true"
                data-testid="canvas-view-mode-live-draft-dot"
              />
            ) : null}
          </span>
        </TabsTrigger>
        {showMemory ? (
          <TabsTrigger value={MEMORY_TAB} data-testid="canvas-view-mode-memory" aria-label="Memory">
            <span className="inline-flex items-center gap-1.5">
              Memory
              {MEMORY_TAB_NAMESPACE_BADGE_ENABLED && memoryNamespaceCount > 0 ? (
                <Badge
                  variant="secondary"
                  className="h-4 px-1.5 py-0 text-[10px] leading-none"
                  data-testid="canvas-view-mode-memory-badge"
                >
                  {memoryNamespaceCount}
                </Badge>
              ) : null}
            </span>
          </TabsTrigger>
        ) : null}
      </TabsList>
    </Tabs>
  );
}
