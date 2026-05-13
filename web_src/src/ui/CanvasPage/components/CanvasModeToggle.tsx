import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

export type CanvasMode = "launchpad" | "version-live" | "runs";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLaunchpad?: () => void;
  onSelectLive: () => void;
  onSelectRuns?: () => void;
  runsNotificationCount?: number;
  /**
   * When true, the active Canvas tab uses an amber palette to signal that the
   * user is currently in edit mode (the canvas being viewed is a draft).
   */
  editing?: boolean;
  /**
   * When true, an amber dot is shown next to the Canvas label to indicate
   * there are unpublished draft changes. Independent of `editing` so the dot
   * can persist after the user exits edit mode without publishing.
   */
  hasDraft?: boolean;
}

export function CanvasModeToggle({
  mode,
  onSelectLaunchpad,
  onSelectLive,
  onSelectRuns,
  runsNotificationCount,
  editing = false,
  hasDraft = false,
}: CanvasModeToggleProps) {
  const handleValueChange = (next: string) => {
    if (next === mode) {
      return;
    }

    if (next === "launchpad" && onSelectLaunchpad) {
      void onSelectLaunchpad();
    } else if (next === "version-live") {
      void onSelectLive();
    } else if (next === "runs" && onSelectRuns) {
      void onSelectRuns();
    }
  };

  // Border-radius on the very first / very last visible trigger gets the
  // "rounded" treatment; all middle triggers stay square. We compute that
  // here so the toggle still looks right when Launchpad and/or Runs are
  // hidden (Live is always present).
  const showLaunchpad = !!onSelectLaunchpad;
  const showRuns = !!onSelectRuns;

  const baseTrigger =
    "border-none px-3 py-1 text-[13px] font-medium text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none";
  // When editing, the active Canvas tab uses an amber/draft palette to make it
  // unambiguous that the canvas being shown represents an in-progress draft.
  const editingActive =
    "data-[state=active]:bg-amber-50 data-[state=active]:text-amber-800 data-[state=active]:ring-1 data-[state=active]:ring-inset data-[state=active]:ring-amber-200";
  const leftRounded = "rounded-sm rounded-br-none rounded-tr-none";
  const rightRounded = "rounded-sm rounded-bl-none rounded-tl-none";
  const middle = "rounded-none";
  const fullRounded = "rounded-sm";

  const launchpadCls = `${baseTrigger} ${leftRounded}`;
  const liveSlotCls =
    showLaunchpad && showRuns ? middle : showLaunchpad ? rightRounded : showRuns ? leftRounded : fullRounded;
  const liveCls = cn(baseTrigger, liveSlotCls, editing && editingActive);
  const runsCls = `${baseTrigger} ${rightRounded}`;

  return (
    <Tabs value={mode} onValueChange={handleValueChange} className="inline-flex w-auto" aria-label="Canvas view">
      <TabsList className="h-7 w-fit gap-0 rounded-sm border border-slate-300 bg-white/80 p-0">
        {showLaunchpad ? (
          <>
            <TabsTrigger
              value="launchpad"
              data-testid="canvas-view-mode-launchpad"
              aria-label="Dashboard"
              className={launchpadCls}
            >
              Dashboard
            </TabsTrigger>
            <div className="h-full w-px bg-slate-300"></div>
          </>
        ) : null}
        <TabsTrigger
          value="version-live"
          data-testid="canvas-view-mode-live"
          aria-label={editing ? "Canvas (editing)" : hasDraft ? "Canvas (unpublished draft)" : "Canvas"}
          className={liveCls}
        >
          <span className="inline-flex items-center gap-1.5">
            Canvas
            {hasDraft ? (
              <span
                className="inline-flex h-1.5 w-1.5 rounded-full bg-amber-500"
                aria-hidden="true"
                data-testid="canvas-view-mode-live-draft-dot"
              />
            ) : null}
          </span>
        </TabsTrigger>
        {showRuns ? (
          <>
            <div className="h-full w-px bg-slate-300"></div>
            <TabsTrigger value="runs" data-testid="canvas-view-mode-runs" aria-label="Runs" className={runsCls}>
              <span className="inline-flex items-center gap-1.5">
                Runs
                {runsNotificationCount && runsNotificationCount > 0 ? (
                  <span className="inline-flex h-4 min-w-[16px] items-center justify-center rounded-full bg-sky-600 px-1 text-[10px] font-medium leading-none text-white">
                    {runsNotificationCount > 99 ? "99+" : runsNotificationCount}
                  </span>
                ) : null}
              </span>
            </TabsTrigger>
          </>
        ) : null}
      </TabsList>
    </Tabs>
  );
}
