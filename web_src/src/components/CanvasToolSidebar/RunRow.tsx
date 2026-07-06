import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "@/ui/Runs/RunNodeIcon";
import { RUN_STATUS_META, type RunStatusKey } from "@/ui/Runs/runPresentation";
import { formatRunDuration } from "@/ui/Runs/runSummary";
import { Link, useParams } from "react-router-dom";
import { isNormalClick } from "@/lib/linkHelpers";

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
  const { organizationId, appId } = useParams<{ organizationId: string; appId: string }>();
  const iconSrc = getHeaderIconSrc(triggerNode?.component);
  const iconSlug = triggerNode?.component ? componentIconMap[triggerNode.component] : undefined;
  const runHref = organizationId && appId && run.id ? appPath(organizationId, appId, `?run=${run.id}`) : "#";
  const duration = formatRunDuration(run);
  const statusMeta = RUN_STATUS_META[status];
  const StatusIcon = statusMeta.icon;

  return (
    <div
      data-testid="runs-sidebar-row"
      className={cn(
        "group relative flex w-full min-w-0 shrink-0 flex-col gap-1 border-b border-b-slate-950/10 px-3 py-2 transition-colors",
        isSelected ? "bg-sky-100" : "hover:bg-gray-50",
      )}
    >
      <Link
        to={runHref}
        onClick={(e) => {
          if (isNormalClick(e)) {
            e.preventDefault();
            if (run.id) onSelectRun(run.id);
          }
        }}
        className="absolute inset-0 z-0"
        aria-label={title}
      />

      {/* Top line: run title + status */}
      <span className="pointer-events-none relative z-0 flex min-w-0 items-center gap-1.5">
        <span
          className={cn(
            "min-w-0 flex-1 truncate text-xs",
            isSelected ? "font-semibold text-sky-900" : "font-medium text-gray-800",
          )}
        >
          {title}
        </span>
        <span
          aria-label={statusMeta.label}
          className={cn(
            "inline-flex shrink-0 items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium ring-1 ring-inset",
            statusMeta.badgeClassName,
          )}
        >
          <StatusIcon className="h-3 w-3" aria-hidden />
          {statusMeta.label}
        </span>
      </span>

      {/* Second line: what triggered the run + timing */}
      <span className="pointer-events-none relative z-0 flex min-w-0 items-center gap-1.5 text-[11px] text-gray-500">
        <RunNodeIcon
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          alt={triggerName}
          size={RUN_NODE_ICON_SIZE}
          className="h-3.5 w-3.5 shrink-0 text-gray-400"
        />
        <span className="min-w-0 truncate" title={triggerName}>
          {triggerName}
        </span>
        <span className="ml-auto flex shrink-0 items-center gap-2 tabular-nums text-gray-500">
          {duration ? <span title="Run duration">{duration}</span> : null}
          {run.createdAt ? <TimeAgo date={run.createdAt} /> : null}
        </span>
      </span>
    </div>
  );
}
