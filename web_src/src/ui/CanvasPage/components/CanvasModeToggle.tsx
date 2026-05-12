import type React from "react";
import { cn } from "@/lib/utils";

type CanvasMode = "version-live" | "version-edit" | "runs";

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
  const showRuns = !!onSelectRuns;
  const baseTrigger =
    "h-full border-none px-3 py-1 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-50";

  return (
    <div className="inline-flex w-auto" aria-label="Canvas view" role="group">
      <div className="flex h-8 w-fit gap-0 overflow-hidden rounded-sm border border-slate-300 bg-white/80 p-0">
        <ModeButton
          isActive={mode === "version-edit"}
          data-testid="canvas-view-mode-editor"
          aria-label="Editor"
          onClick={() => {
            if (mode !== "version-edit") void onSelectEditor();
          }}
          className={cn(baseTrigger, "rounded-sm rounded-br-none rounded-tr-none")}
        >
          Editor
        </ModeButton>
        <div className="h-full w-px bg-slate-300"></div>
        <ModeButton
          isActive={mode === "version-live"}
          data-testid="canvas-view-mode-live"
          aria-label="Live Canvas"
          onClick={() => {
            if (mode !== "version-live") void onSelectLive();
          }}
          className={cn(baseTrigger, showRuns ? "rounded-none" : "rounded-sm rounded-bl-none rounded-tl-none")}
        >
          Live Canvas
        </ModeButton>
        {showRuns ? (
          <>
            <div className="h-full w-px bg-slate-300"></div>
            <ModeButton
              isActive={mode === "runs"}
              data-testid="canvas-view-mode-runs"
              aria-label="Runs"
              onClick={() => {
                if (mode !== "runs" && onSelectRuns) void onSelectRuns();
              }}
              className={cn(baseTrigger, "rounded-sm rounded-bl-none rounded-tl-none")}
            >
              <span className="inline-flex items-center gap-1.5">
                Runs
                {runsNotificationCount && runsNotificationCount > 0 ? (
                  <span className="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-sky-600 px-1 text-[10px] font-medium leading-none text-white">
                    {runsNotificationCount > 99 ? "99+" : runsNotificationCount}
                  </span>
                ) : null}
              </span>
            </ModeButton>
          </>
        ) : null}
      </div>
    </div>
  );
}

interface ModeButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  isActive: boolean;
}

function ModeButton({ isActive, className, ...props }: ModeButtonProps) {
  return (
    <button
      type="button"
      aria-pressed={isActive}
      className={cn(isActive && "bg-sky-50 text-sky-700 shadow-none", className)}
      {...props}
    />
  );
}
