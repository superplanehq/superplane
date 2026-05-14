import { Rabbit } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { RUN_STATUS_META, type RunStatusKey } from "@/ui/Runs/runPresentation";

interface RunChipProps {
  runId: string;
  label: string;
  status: RunStatusKey;
  canvasId: string;
  organizationId: string;
}

function parseStatus(raw?: string): RunStatusKey {
  if (!raw) return "unknown";
  const s = raw.toLowerCase();
  if (s === "passed" || s === "success") return "passed";
  if (s === "failed" || s === "error" || s === "failure") return "failed";
  if (s === "running" || s === "started") return "running";
  if (s === "cancelled") return "cancelled";
  return "unknown";
}

export function RunChipFromLink({
  runId,
  rawLabel,
  rawStatus,
  canvasId,
  organizationId,
}: {
  runId: string;
  rawLabel?: string;
  rawStatus?: string;
  canvasId: string;
  organizationId: string;
}) {
  const status = parseStatus(rawStatus);
  const label = rawLabel && rawLabel !== "run" ? rawLabel : `#${runId.substring(0, 8)}`;
  return <RunChip runId={runId} label={label} status={status} canvasId={canvasId} organizationId={organizationId} />;
}

export function RunChip({ runId, label, status, canvasId, organizationId }: RunChipProps) {
  const navigate = useNavigate();
  const meta = RUN_STATUS_META[status];

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
