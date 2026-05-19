import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunNodeIcon } from "@/ui/Runs/RunNodeIcon";
import { RUN_STATUS_META, type RunStatusKey } from "@/ui/Runs/runPresentation";
import { Link as LinkIcon } from "lucide-react";
import { toast } from "sonner";

interface RunRowProps {
  run: CanvasesCanvasRun;
  triggerName: string;
  title: string;
  status: RunStatusKey;
  triggerNode?: ComponentsNode;
  isSelected: boolean;
  componentIconMap: Record<string, string>;
  onSelectRun: (runId: string) => void;
}

export function RunRow({
  run,
  triggerName,
  title,
  status,
  triggerNode,
  isSelected,
  componentIconMap,
  onSelectRun,
}: RunRowProps) {
  const iconSrc = getHeaderIconSrc(triggerNode?.component);
  const iconSlug = triggerNode?.component ? componentIconMap[triggerNode.component] : undefined;

  return (
    <div
      data-testid="runs-sidebar-row"
      role="button"
      tabIndex={0}
      onClick={() => run.id && onSelectRun(run.id)}
      onKeyDown={(event) => {
        if (event.key !== "Enter" && event.key !== " ") return;
        event.preventDefault();
        if (run.id) {
          onSelectRun(run.id);
        }
      }}
      className={cn(
        "group flex w-full cursor-pointer items-center gap-1.5 border-b border-l-2 border-slate-100 px-3 py-2 text-left transition-colors",
        status === "failed" ? "border-l-red-400" : "border-l-transparent",
        isSelected ? "border-l-sky-500 bg-sky-100" : "hover:bg-gray-50",
      )}
    >
      <RunNodeIcon
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        alt={triggerName}
        size={14}
        className="shrink-0 text-gray-400"
      />
      <span
        className={cn(
          "max-w-[35%] shrink-0 truncate rounded px-1.5 py-0.5 text-[10px] font-medium",
          isSelected ? "bg-sky-200 text-sky-800" : "bg-slate-100 text-slate-600",
        )}
      >
        {triggerName}
      </span>
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-xs",
          isSelected ? "font-semibold text-sky-900" : "font-medium text-gray-800",
        )}
      >
        {title}
      </span>
      <span
        aria-label={RUN_STATUS_META[status].label}
        title={RUN_STATUS_META[status].label}
        className={cn("inline-block h-2 w-2 shrink-0 rounded-full", RUN_STATUS_META[status].dotClassName)}
      />
      <button
        type="button"
        title="Copy link to run"
        className="hidden shrink-0 rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 group-hover:inline-flex"
        onClick={(event) => {
          event.stopPropagation();
          const url = new URL(window.location.href);
          url.searchParams.set("view", "runs");
          url.searchParams.set("run", run.id || "");
          navigator.clipboard.writeText(url.toString());
          toast.success("Run link copied");
        }}
      >
        <LinkIcon className="h-3 w-3" />
      </button>
      {run.createdAt ? (
        <span className="shrink-0 text-[10px] tabular-nums text-gray-400">
          <TimeAgo date={run.createdAt} />
        </span>
      ) : null}
    </div>
  );
}
