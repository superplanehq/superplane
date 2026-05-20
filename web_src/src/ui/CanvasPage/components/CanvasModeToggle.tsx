import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

type CanvasMode = "version-live" | "version-edit" | "runs" | "dashboard";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLive: () => void;
  onSelectDashboard?: () => void;
  editing?: boolean;
  hasDraft?: boolean;
}

const CANVAS_TAB = "canvas";
const DASHBOARD_TAB = "dashboard";

export function CanvasModeToggle({
  mode,
  onSelectLive,
  onSelectDashboard,
  editing = false,
  hasDraft = false,
}: CanvasModeToggleProps) {
  const showDashboard = Boolean(onSelectDashboard);
  const selected = mode === DASHBOARD_TAB ? DASHBOARD_TAB : CANVAS_TAB;

  return (
    <Tabs
      value={selected}
      onValueChange={(next) => {
        if (next === CANVAS_TAB && selected !== CANVAS_TAB) void onSelectLive();
        if (next === DASHBOARD_TAB && selected !== DASHBOARD_TAB && onSelectDashboard) void onSelectDashboard();
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
      </TabsList>
    </Tabs>
  );
}
