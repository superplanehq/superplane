import { Rabbit } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import type { CanvasesCanvasRun } from "@/api-client";
import { RUN_STATUS_META, getRunStatus, shortId } from "@/ui/Runs/runPresentation";

interface RunChipProps {
  runId: string;
  canvasId: string;
  organizationId: string;
}

function findRunInCache(queryClient: ReturnType<typeof useQueryClient>, runId: string): CanvasesCanvasRun | undefined {
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
  const status = run ? getRunStatus(run) : "unknown";
  const meta = RUN_STATUS_META[status];
  const label = `#${shortId(runId)}`;

  return (
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
  );
}
