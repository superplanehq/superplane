import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { Pencil } from "lucide-react";

export type CanvasMode = "launchpad" | "version-live" | "runs";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLaunchpad?: () => void;
  onSelectLive: () => void;
  onSelectRuns?: () => void;
  runsNotificationCount?: number;
  /**
   * When true, the Canvas tab is highlighted as a draft/edit state. Used while
   * the user is in version-edit mode so the toggle stays visible (for parking
   * lot navigation) but signals that the canvas tab represents the draft.
   */
  editing?: boolean;
}

export function CanvasModeToggle({
  mode,
  onSelectLaunchpad,
  onSelectLive,
  onSelectRuns,
  runsNotificationCount,
  editing = false,
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
    "border-none px-3 py-1 text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none";
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
      <TabsList className="h-8 w-fit gap-0 rounded-sm border border-slate-300 bg-white/80 p-0">
        {showLaunchpad ? (
          <>
            <TabsTrigger
              value="launchpad"
              data-testid="canvas-view-mode-launchpad"
              aria-label="Apps"
              className={launchpadCls}
            >
              Apps
            </TabsTrigger>
            <div className="h-full w-px bg-slate-300"></div>
          </>
        ) : null}
        <TabsTrigger
          value="version-live"
          data-testid="canvas-view-mode-live"
          aria-label={editing ? "Canvas (editing)" : "Canvas"}
          className={liveCls}
        >
          <span className="inline-flex items-center gap-1.5">
            {editing ? <Pencil className="h-3 w-3" aria-hidden="true" /> : null}
            Canvas
            {editing ? <span className="inline-flex h-1.5 w-1.5 rounded-full bg-amber-500" aria-hidden="true" /> : null}
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
