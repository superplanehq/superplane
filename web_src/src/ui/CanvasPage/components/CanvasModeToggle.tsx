import type React from "react";
import { cn } from "@/lib/utils";

type CanvasMode = "version-live" | "version-edit" | "runs" | "dashboard";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectLive: () => void;
  onSelectRuns?: () => void;
  onSelectDashboard?: () => void;
  runsNotificationCount?: number;
  editing?: boolean;
  hasDraft?: boolean;
}

export function CanvasModeToggle({
  mode,
  onSelectLive,
  onSelectRuns,
  onSelectDashboard,
  runsNotificationCount,
  editing = false,
  hasDraft = false,
}: CanvasModeToggleProps) {
  const showRuns = !!onSelectRuns;
  const showDashboard = !!onSelectDashboard;
  const baseTrigger =
    "h-full border-none px-3 py-1 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-50";
  const canvasActiveClassName =
    editing || hasDraft
      ? "bg-amber-50 text-amber-800 shadow-none ring-1 ring-inset ring-amber-200"
      : "bg-sky-50 text-sky-700 shadow-none";
  const canvasShapeClassName = getCanvasShapeClassName(showDashboard, showRuns);

  return (
    <div className="inline-flex w-auto" aria-label="Canvas view" role="group">
      <div className="flex h-8 w-fit gap-0 overflow-hidden rounded-sm border border-slate-300 bg-white/80 p-0">
        {showDashboard && onSelectDashboard ? (
          <DashboardModeTab mode={mode} onSelectDashboard={onSelectDashboard} baseTrigger={baseTrigger} />
        ) : null}
        <ModeButton
          isActive={mode === "version-live" || mode === "version-edit"}
          activeClassName={canvasActiveClassName}
          data-testid="canvas-view-mode-live"
          aria-label={editing ? "Canvas (editing)" : hasDraft ? "Canvas (unpublished draft)" : "Canvas"}
          onClick={() => {
            if (mode !== "version-live" && mode !== "version-edit") void onSelectLive();
          }}
          className={cn(baseTrigger, canvasShapeClassName)}
        >
          <span className="inline-flex items-center gap-1.5">
            Canvas
            {hasDraft ? (
              <span
                className="inline-flex h-1.5 w-1.5 rounded-full bg-orange-500"
                aria-hidden="true"
                data-testid="canvas-view-mode-live-draft-dot"
              />
            ) : null}
          </span>
        </ModeButton>
        {showRuns && onSelectRuns ? (
          <RunsModeTab
            mode={mode}
            onSelectRuns={onSelectRuns}
            runsNotificationCount={runsNotificationCount}
            baseTrigger={baseTrigger}
          />
        ) : null}
      </div>
    </div>
  );
}

function getCanvasShapeClassName(showDashboard: boolean, showRuns: boolean) {
  if (showDashboard && showRuns) return "rounded-none";
  if (showDashboard) return "rounded-l-none rounded-r-sm";
  if (showRuns) return "rounded-l-sm rounded-r-none";
  return "rounded-sm";
}

function DashboardModeTab({
  mode,
  onSelectDashboard,
  baseTrigger,
}: {
  mode: CanvasMode;
  onSelectDashboard: () => void;
  baseTrigger: string;
}) {
  return (
    <>
      <ModeButton
        isActive={mode === "dashboard"}
        data-testid="canvas-view-mode-dashboard"
        aria-label="Dashboard"
        onClick={() => {
          if (mode !== "dashboard") void onSelectDashboard();
        }}
        className={cn(baseTrigger, "rounded-l-sm rounded-r-none")}
      >
        Dashboard
      </ModeButton>
      <div className="h-full w-px bg-slate-300" />
    </>
  );
}

function RunsModeTab({
  mode,
  onSelectRuns,
  runsNotificationCount,
  baseTrigger,
}: {
  mode: CanvasMode;
  onSelectRuns: () => void;
  runsNotificationCount?: number;
  baseTrigger: string;
}) {
  return (
    <>
      <div className="h-full w-px bg-slate-300" />
      <ModeButton
        isActive={mode === "runs"}
        data-testid="canvas-view-mode-runs"
        aria-label="Runs"
        onClick={() => {
          if (mode !== "runs") void onSelectRuns();
        }}
        className={cn(baseTrigger, "rounded-l-none rounded-r-sm")}
      >
        <span className="inline-flex items-center gap-1.5">
          Runs
          {runsNotificationCount != null && runsNotificationCount > 0 ? (
            <span className="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-sky-600 px-1 text-[10px] font-medium leading-none text-white">
              {runsNotificationCount > 99 ? "99+" : runsNotificationCount}
            </span>
          ) : null}
        </span>
      </ModeButton>
    </>
  );
}

interface ModeButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  isActive: boolean;
  activeClassName?: string;
}

function ModeButton({
  isActive,
  activeClassName = "bg-sky-50 text-sky-700 shadow-none",
  className,
  ...props
}: ModeButtonProps) {
  return (
    <button type="button" aria-pressed={isActive} className={cn(isActive && activeClassName, className)} {...props} />
  );
}
