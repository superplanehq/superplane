import { cn } from "@/lib/utils";
import { formatDuration } from "@/lib/duration";

export type RibbonStepStatus = "success" | "error" | "running" | "queued" | "cancelled";

export interface RibbonStep {
  key: string;
  name: string;
  status: RibbonStepStatus;
  isTrigger: boolean;
  durationMs: number;
  finished: boolean;
}

interface StepRibbonProps {
  steps: RibbonStep[];
  totalDurationMs: number;
  onStepClick?: (key: string) => void;
}

const BAR_COLOR: Record<RibbonStepStatus, string> = {
  success: "bg-emerald-500",
  error: "bg-red-500",
  running: "bg-amber-500",
  queued: "bg-gray-300",
  cancelled: "bg-slate-400",
};

function statusLabel(status: RibbonStepStatus): string {
  switch (status) {
    case "success":
      return "Succeeded";
    case "error":
      return "Failed";
    case "running":
      return "Running";
    case "queued":
      return "Queued";
    case "cancelled":
      return "Cancelled";
  }
}

function buildCaption(steps: RibbonStep[], totalDurationMs: number): string {
  const execSteps = steps.filter((s) => !s.isTrigger);
  if (execSteps.length === 0) return "No steps executed yet";

  const running = execSteps.filter((s) => s.status === "running" || s.status === "queued").length;
  const passed = execSteps.filter((s) => s.status === "success").length;
  const failed = execSteps.filter((s) => s.status === "error").length;
  const cancelled = execSteps.filter((s) => s.status === "cancelled").length;

  const durationPart = totalDurationMs > 0 ? formatDuration(totalDurationMs) : null;

  if (running > 0) {
    const base = `${running} of ${execSteps.length} running`;
    return durationPart ? `${base} · elapsed ${durationPart}` : base;
  }

  const parts: string[] = [];
  parts.push(`${execSteps.length} ${execSteps.length === 1 ? "step" : "steps"}`);
  if (passed > 0) parts.push(`${passed} passed`);
  if (failed > 0) parts.push(`${failed} failed`);
  if (cancelled > 0) parts.push(`${cancelled} cancelled`);
  const left = parts.join(", ");
  return durationPart ? `${left} · ${durationPart}` : left;
}

export function StepRibbon({ steps, totalDurationMs, onStepClick }: StepRibbonProps) {
  if (steps.length === 0) return null;

  const caption = buildCaption(steps, totalDurationMs);

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex h-2 w-full items-stretch gap-[2px]">
        {steps.map((step) => {
          const isActive = step.status === "running";
          return (
            <button
              key={step.key}
              type="button"
              onClick={() => onStepClick?.(step.key)}
              title={`${step.name} · ${statusLabel(step.status)}${
                step.durationMs > 0 ? ` · ${formatDuration(step.durationMs)}` : ""
              }`}
              aria-label={`${step.name}: ${statusLabel(step.status)}`}
              className={cn(
                "group relative h-full flex-1 overflow-hidden rounded-[2px] transition-transform",
                BAR_COLOR[step.status],
                step.isTrigger && "max-w-[10px]",
                "hover:scale-y-[1.4]",
              )}
            >
              {isActive ? (
                <span className="absolute inset-0 animate-pulse bg-white/30" aria-hidden />
              ) : null}
            </button>
          );
        })}
      </div>
      <div className="text-xs text-gray-500">{caption}</div>
    </div>
  );
}
