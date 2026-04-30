import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

export type CanvasMode = "launchpad" | "version-live" | "version-edit" | "runs";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLaunchpad?: () => void;
  onSelectEditor: () => void;
  onSelectLive: () => void;
  onSelectRuns?: () => void;
  runsNotificationCount?: number;
}

export function CanvasModeToggle({
  mode,
  onSelectLaunchpad,
  onSelectEditor,
  onSelectLive,
  onSelectRuns,
  runsNotificationCount,
}: CanvasModeToggleProps) {
  const handleValueChange = (next: string) => {
    if (next === mode) {
      return;
    }

    if (next === "launchpad" && onSelectLaunchpad) {
      void onSelectLaunchpad();
    } else if (next === "version-edit") {
      void onSelectEditor();
    } else if (next === "version-live") {
      void onSelectLive();
    } else if (next === "runs" && onSelectRuns) {
      void onSelectRuns();
    }
  };

  // Border-radius on the very first / very last visible trigger gets the
  // "rounded" treatment; all middle triggers stay square. We compute that here
  // so the toggle still looks right when Launchpad and/or Runs are hidden.
  const showLaunchpad = !!onSelectLaunchpad;
  const showRuns = !!onSelectRuns;

  const baseTrigger =
    "border-none px-3 py-1 text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none";
  const leftRounded = "rounded-sm rounded-br-none rounded-tr-none";
  const rightRounded = "rounded-sm rounded-bl-none rounded-tl-none";
  const middle = "rounded-none";

  const launchpadCls = `${baseTrigger} ${leftRounded}`;
  const editorCls = `${baseTrigger} ${showLaunchpad ? middle : leftRounded}`;
  const liveCls = `${baseTrigger} ${showRuns ? middle : rightRounded}`;
  const runsCls = `${baseTrigger} ${rightRounded}`;

  return (
    <Tabs value={mode} onValueChange={handleValueChange} className="inline-flex w-auto" aria-label="Canvas view">
      <TabsList className="h-8 w-fit gap-0 rounded-sm border border-slate-300 bg-white/80 p-0">
        {showLaunchpad ? (
          <>
            <TabsTrigger
              value="launchpad"
              data-testid="canvas-view-mode-launchpad"
              aria-label="Launchpad"
              className={launchpadCls}
            >
              Launchpad
            </TabsTrigger>
            <div className="h-full w-px bg-slate-300"></div>
          </>
        ) : null}
        <TabsTrigger
          value="version-edit"
          data-testid="canvas-view-mode-editor"
          aria-label="Editor"
          className={editorCls}
        >
          Editor
        </TabsTrigger>
        <div className="h-full w-px bg-slate-300"></div>
        <TabsTrigger value="version-live" data-testid="canvas-view-mode-live" aria-label="Live" className={liveCls}>
          Live
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
