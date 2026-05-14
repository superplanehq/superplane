import { Rabbit } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";

type RunStatus = "passed" | "failed" | "running" | "unknown";

interface RunChipProps {
  runId: string;
  canvasId: string;
  organizationId: string;
  label?: string;
  status?: RunStatus;
}

const STATUS_STYLES: Record<RunStatus, string> = {
  passed: "bg-green-100 text-green-700 hover:bg-green-200",
  failed: "bg-red-100 text-red-700 hover:bg-red-200",
  running: "bg-amber-100 text-amber-700 hover:bg-amber-200",
  unknown: "bg-violet-100 text-violet-700 hover:bg-violet-200",
};

function parseStatus(raw?: string): RunStatus {
  if (!raw) return "unknown";
  const s = raw.toLowerCase();
  if (s === "passed" || s === "success" || s === "ok") return "passed";
  if (s === "failed" || s === "error" || s === "failure") return "failed";
  if (s === "running" || s === "in_progress" || s === "pending") return "running";
  return "unknown";
}

function truncate(text: string, max: number): string {
  return text.length > max ? text.substring(0, max) + "…" : text;
}

export function RunChip({ runId, canvasId, organizationId, label, status: rawStatus }: RunChipProps) {
  const navigate = useNavigate();
  const status = parseStatus(rawStatus);
  const displayLabel = label ? truncate(label, 30) : `#${runId.substring(0, 6)}`;

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
      {displayLabel}
    </button>
  );
}
