import { Rabbit } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import type { CanvasesCanvasRun } from "@/api-client";
import { canvasesListRuns } from "@/api-client";
import { RUN_STATUS_META, getRunStatus, shortId } from "@/ui/Runs/runPresentation";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";

interface RunChipProps {
  runId: string;
  canvasId: string;
  organizationId: string;
}

function findRunInCache(queryClient: ReturnType<typeof useQueryClient>, runId: string): CanvasesCanvasRun | undefined {
  // Search all canvas run query caches
  const queries = queryClient.getQueriesData<{ pages?: Array<{ runs?: CanvasesCanvasRun[] }> }>({
    queryKey: ["canvas", "runs"],
  });
  for (const [, data] of queries) {
    if (!data?.pages) continue;
    for (const page of data.pages) {
      const found = page?.runs?.find((r) => r.id === runId);
      if (found) return found;
    }
  }

  // Also check our own chip cache
  const chipData = queryClient.getQueryData<{ runs?: CanvasesCanvasRun[] }>(["agent-run-chips"]);
  return chipData?.runs?.find((r) => r.id === runId);
}

function useRunData(canvasId: string, runId: string) {
  const queryClient = useQueryClient();
  const cached = findRunInCache(queryClient, runId);

  // Fetch runs if not in any cache — one shared query for all chips
  const { data } = useQuery({
    queryKey: ["agent-run-chips", canvasId],
    queryFn: async () => {
      const response = await canvasesListRuns({
        path: { canvasId },
        query: { limit: 50 },
        headers: { "x-organization-id": "" }, // filled by interceptor
      });
      return response.data ?? { runs: [] };
    },
    enabled: !cached,
    staleTime: 60_000,
  });

  if (cached) return cached;
  return data?.runs?.find((r: CanvasesCanvasRun) => r.id === runId);
}

function formatDuration(start?: string, end?: string): string {
  if (!start) return "—";
  const s = new Date(start).getTime();
  const e = end ? new Date(end).getTime() : Date.now();
  const ms = e - s;
  if (ms < 1000) return `${ms}ms`;
  const sec = Math.round(ms / 1000);
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  return `${min}m ${sec % 60}s`;
}

function formatTime(date?: string): string {
  if (!date) return "—";
  return new Date(date).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

export function RunChip({ runId, canvasId, organizationId }: RunChipProps) {
  const navigate = useNavigate();
  const run = useRunData(canvasId, runId);
  const status = run ? getRunStatus(run) : "unknown";
  const meta = RUN_STATUS_META[status];
  const StatusIcon = meta.icon;
  const label = `#${shortId(runId)}`;

  return (
    <HoverCard openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          onClick={() => navigate(`/${organizationId}/canvases/${canvasId}?view=runs&run=${runId}`)}
          className={cn(
            "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ring-1 ring-inset transition-colors cursor-pointer align-middle",
            meta.badgeClassName,
          )}
          title={`Run ${runId}`}
        >
          <Rabbit className="size-3" />
          {label}
        </button>
      </HoverCardTrigger>
      <HoverCardContent className="w-72 p-0" side="top" align="start">
        <RunHoverContent run={run} runId={runId} status={status} meta={meta} StatusIcon={StatusIcon} />
      </HoverCardContent>
    </HoverCard>
  );
}

function RunHoverContent({
  run,
  runId,
  status,
  meta,
  StatusIcon,
}: {
  run?: CanvasesCanvasRun;
  runId: string;
  status: string;
  meta: (typeof RUN_STATUS_META)[keyof typeof RUN_STATUS_META];
  StatusIcon: React.ComponentType<{ className?: string }>;
}) {
  if (!run) {
    return (
      <div className="p-3 text-xs text-slate-500">
        Run <span className="font-mono">{shortId(runId)}</span> — loading...
      </div>
    );
  }

  const duration = formatDuration(run.createdAt, run.finishedAt);
  const executions = run.executions ?? [];

  return (
    <div>
      <div className="flex items-center gap-2 px-3 py-2 border-b border-slate-100">
        <StatusIcon className="size-4 shrink-0" />
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium text-slate-900 truncate">Run #{shortId(runId)}</p>
          <p className="text-[10px] text-slate-500">{meta.label} · {duration}</p>
        </div>
        <span className={cn("inline-block h-2 w-2 shrink-0 rounded-full", meta.dotClassName)} />
      </div>

      <div className="px-3 py-2 border-b border-slate-100 grid grid-cols-2 gap-x-4 gap-y-1 text-[10px]">
        <div>
          <span className="text-slate-400">Started</span>
          <p className="text-slate-700 font-medium">{formatTime(run.createdAt)}</p>
        </div>
        <div>
          <span className="text-slate-400">Finished</span>
          <p className="text-slate-700 font-medium">{run.finishedAt ? formatTime(run.finishedAt) : "—"}</p>
        </div>
      </div>

      {executions.length > 0 && (
        <div className="px-3 py-2">
          <p className="text-[10px] text-slate-400 mb-1">{executions.length} node{executions.length !== 1 ? "s" : ""} executed</p>
          <div className="space-y-0.5">
            {executions.slice(0, 6).map((exec, i) => (
              <div key={i} className="flex items-center gap-1.5 text-[10px]">
                <span
                  className={cn(
                    "inline-block h-1.5 w-1.5 shrink-0 rounded-full",
                    exec.result === "RESULT_PASSED"
                      ? "bg-emerald-500"
                      : exec.result === "RESULT_FAILED"
                        ? "bg-red-500"
                        : exec.state === "STATE_STARTED"
                          ? "bg-blue-500 animate-pulse"
                          : "bg-slate-300",
                  )}
                />
                <span className="text-slate-600 truncate">{exec.nodeId ? shortId(exec.nodeId) : `Step ${i + 1}`}</span>
              </div>
            ))}
            {executions.length > 6 && (
              <p className="text-[10px] text-slate-400">+{executions.length - 6} more</p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
