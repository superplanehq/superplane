import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "@/ui/Runs/RunNodeIcon";
import { RUN_STATUS_META, type RunStatusKey } from "@/ui/Runs/runPresentation";
import { Link as LinkIcon } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { toast } from "sonner";

interface RunRowProps {
  run: CanvasesCanvasRun;
  triggerName: string;
  title: string;
  status: RunStatusKey;
  triggerNode?: ComponentsNode;
  isSelected: boolean;
  hideBottomBorder?: boolean;
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
  hideBottomBorder = false,
  componentIconMap,
  onSelectRun,
}: RunRowProps) {
  const { organizationId, appId } = useParams<{ organizationId: string; appId: string }>();
  const iconSrc = getHeaderIconSrc(triggerNode?.component);
  const iconSlug = triggerNode?.component ? componentIconMap[triggerNode.component] : undefined;
  const runHref = organizationId && appId && run.id ? appPath(organizationId, appId, `?view=runs&run=${run.id}`) : "#";

  return (
    <Link
      to={runHref}
      data-testid="runs-sidebar-row"
      onClick={() => {
        if (run.id) onSelectRun(run.id);
      }}
      className={cn(
        "group flex w-full cursor-pointer items-center gap-1.5 px-3 py-2 text-left transition-colors",
        !hideBottomBorder && "border-b border-b-slate-950/10",
        isSelected ? "bg-sky-100" : "hover:bg-gray-50",
      )}
    >
      <RunNodeIcon
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        alt={triggerName}
        size={RUN_NODE_ICON_SIZE}
        className={cn("h-3.5 w-3.5 shrink-0", isSelected ? "text-gray-800" : "text-gray-500")}
      />
      <span
        aria-label={RUN_STATUS_META[status].label}
        title={RUN_STATUS_META[status].label}
        className={cn("inline-block h-2 w-2 shrink-0 rounded-full", RUN_STATUS_META[status].dotClassName)}
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
        role="button"
        tabIndex={0}
        title="Copy link to run"
        className="hidden shrink-0 rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 group-hover:inline-flex"
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            event.stopPropagation();
            void (async () => {
              const url = new URL(window.location.href);
              url.searchParams.set("view", "runs");
              url.searchParams.set("run", run.id || "");
              try {
                await navigator.clipboard.writeText(url.toString());
                toast.success("Run link copied");
              } catch {
                toast.error("Failed to copy run link");
              }
            })();
          }
        }}
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          void (async () => {
            const copyUrl = new URL(runHref, window.location.origin);
            try {
              await navigator.clipboard.writeText(copyUrl.toString());
              toast.success("Run link copied");
            } catch {
              toast.error("Failed to copy run link");
            }
          })();
        }}
      >
        <LinkIcon className="h-3 w-3" />
      </span>
      {run.createdAt ? (
        <span className="shrink-0 text-xs tabular-nums text-gray-500">
          <TimeAgo date={run.createdAt} includeAgo={false} />
        </span>
      ) : null}
    </Link>
  );
}
