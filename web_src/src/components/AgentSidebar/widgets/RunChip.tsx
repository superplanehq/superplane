import { Rabbit } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import type { CanvasesCanvasRun } from "@/api-client";

interface RunChipProps {
  runId: string;
  canvasId: string;
  organizationId: string;
}

type ChipStatus = "passed" | "failed" | "running" | "unknown";

const STATUS_STYLES: Record<ChipStatus, string> = {
  passed: "bg-green-100 text-green-700 hover:bg-green-200",
  failed: "bg-red-100 text-red-700 hover:bg-red-200",
  running: "bg-amber-100 text-amber-700 hover:bg-amber-200",
  unknown: "bg-violet-100 text-violet-700 hover:bg-violet-200",
};

function resolveStatus(run?: CanvasesCanvasRun): ChipStatus {
  if (!run) return "unknown";
  if (run.state === "STATE_STARTED") return "running";
  if (run.result === "RESULT_PASSED") return "passed";
  if (run.result === "RESULT_FAILED") return "failed";
  if (run.result === "RESULT_CANCELLED") return "failed";
  return "unknown";
}

function resolveLabel(run?: CanvasesCanvasRun, runId?: string): string {
  const shortId = `#${(runId ?? run?.id ?? "").substring(0, 6)}`;
  if (!run) return shortId;

  // Use trigger event name or short ID
  const trigger = run.rootEvent?.component;
  if (trigger) return `${shortId} ${trigger}`;
  return shortId;
}

function findRunInCache(queryClient: ReturnType<typeof useQueryClient>, runId: string): CanvasesCanvasRun | undefined {
  // Search through all cached canvas run queries
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
  return undefined;
}

export function RunChip({ runId, canvasId, organizationId }: RunChipProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const run = findRunInCache(queryClient, runId);
  const status = resolveStatus(run);
  const label = resolveLabel(run, runId);

  return (
    <button
      type="button"
      onClick={() => navigate(`/${organizationId}/canvases/${canvasId}?view=runs&run=${runId}`)}
      className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-colors cursor-pointer align-middle",
        STATUS_STYLES[status],
      )}
      title={`Run ${runId}`}
    >
      <Rabbit className="size-3" />
      {label}
    </button>
  );
}
