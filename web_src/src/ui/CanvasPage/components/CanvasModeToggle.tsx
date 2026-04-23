import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

export type CanvasMode = "version-live" | "version-edit" | "runs";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectEditor: () => void;
  onSelectLive: () => void;
  onSelectRuns?: () => void;
  runsNotificationCount?: number;
}

export function CanvasModeToggle({
  mode,
  onSelectEditor,
  onSelectLive,
  onSelectRuns,
  runsNotificationCount,
}: CanvasModeToggleProps) {
  const handleValueChange = (next: string) => {
    if (next === mode) {
      return;
    }

    if (next === "version-edit") {
      void onSelectEditor();
    } else if (next === "version-live") {
      void onSelectLive();
    } else if (next === "runs" && onSelectRuns) {
      void onSelectRuns();
    }
  };

  return (
    <Tabs value={mode} onValueChange={handleValueChange} className="inline-flex w-auto" aria-label="Canvas view">
      <TabsList className="h-8 w-fit gap-0 rounded-sm border border-slate-300 bg-white/80 p-0">
        <TabsTrigger
          value="version-edit"
          data-testid="canvas-view-mode-editor"
          aria-label="Editor"
          className="rounded-sm rounded-br-none rounded-tr-none border-none px-3 py-1 text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none"
        >
          Editor
        </TabsTrigger>
        <div className="h-full w-px bg-slate-300"></div>
        <TabsTrigger
          value="version-live"
          data-testid="canvas-view-mode-live"
          aria-label="Live"
          className="rounded-none border-none px-3 py-1 text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none"
        >
          Live
        </TabsTrigger>
        {onSelectRuns ? (
          <>
            <div className="h-full w-px bg-slate-300"></div>
            <TabsTrigger
              value="runs"
              data-testid="canvas-view-mode-runs"
              aria-label="Runs"
              className="rounded-sm rounded-bl-none rounded-tl-none border-none px-3 py-1 text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none"
            >
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
