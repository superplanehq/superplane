import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

type CanvasMode = "version-live" | "version-edit" | "runs";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLive: () => void;
  onSelectRuns?: () => void;
  runsNotificationCount?: number;
  editing?: boolean;
  hasDraft?: boolean;
}

const CANVAS_TAB = "canvas";
const RUNS_TAB = "runs";

export function CanvasModeToggle({
  mode,
  onSelectLive,
  onSelectRuns,
  runsNotificationCount,
  editing = false,
  hasDraft = false,
}: CanvasModeToggleProps) {
  const showRuns = Boolean(onSelectRuns);
  const selected = mode === RUNS_TAB ? RUNS_TAB : CANVAS_TAB;

  return (
    <Tabs
      value={selected}
      onValueChange={(next) => {
        if (next === CANVAS_TAB && selected !== CANVAS_TAB) void onSelectLive();
        if (next === RUNS_TAB && selected !== RUNS_TAB && onSelectRuns) void onSelectRuns();
      }}
    >
      <TabsList aria-label="Canvas view" className="h-8 min-h-8 bg-slate-100 [&_[data-slot=tabs-trigger]]:text-[13px]">
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
        {showRuns ? (
          <TabsTrigger value={RUNS_TAB} data-testid="canvas-view-mode-runs" aria-label="Runs">
            <span className="inline-flex items-center gap-1.5">
              Runs
              {runsNotificationCount != null && runsNotificationCount > 0 ? (
                <span className="text-muted-foreground tabular-nums text-[13px] leading-none">
                  {runsNotificationCount > 99 ? "99+" : runsNotificationCount}
                </span>
              ) : null}
            </span>
          </TabsTrigger>
        ) : null}
      </TabsList>
    </Tabs>
  );
}
