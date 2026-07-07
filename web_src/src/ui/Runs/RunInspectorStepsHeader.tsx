import { RUN_STATUS_META } from "./runPresentation";
import { cn } from "@/lib/utils";

export function RunInspectorStepsHeader({
  status,
  errorCount,
  stepCount,
}: {
  status: keyof typeof RUN_STATUS_META;
  errorCount: number;
  stepCount: number;
}) {
  const statusMeta = RUN_STATUS_META[status];
  const label = errorCount > 0 ? `Errors ${errorCount}` : statusMeta.label;
  const dotClassName = errorCount > 0 ? "bg-red-500" : statusMeta.dotClassName;

  return (
    <div className="sticky top-0 z-10 flex items-center gap-2 border-b border-slate-950/10 bg-white/95 px-4 py-2 text-xs font-semibold uppercase tracking-wide text-slate-500 backdrop-blur dark:border-gray-800 dark:bg-gray-950/95 dark:text-gray-400">
      <span>Steps</span>
      <span className="ml-2 inline-flex items-center gap-2 font-medium normal-case tracking-normal text-slate-500 dark:text-gray-400">
        <span className={cn("h-2 w-2 rounded-full", dotClassName)} />
        {label}
      </span>
      <span className="sr-only">{stepCount} total steps</span>
    </div>
  );
}
