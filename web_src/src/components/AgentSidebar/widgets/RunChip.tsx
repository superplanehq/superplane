import { Rabbit } from "lucide-react";
import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";
import { appPath } from "@/lib/appPaths";
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

/** Chat-local color overrides for run pills (e.g. green instead of emerald for passed). */
const STYLE_BY_STATUS: Partial<Record<RunStatusKey, string>> = {
  passed: "bg-green-50 text-green-700",
};

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
  const meta = RUN_STATUS_META[status];

  return (
    <Link
      to={appPath(organizationId, canvasId, `?run=${runId}`)}
      className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 rounded-full ring-0 text-xs font-medium transition-colors cursor-pointer align-middle",
        meta.badgeClassName,
        STYLE_BY_STATUS[status],
      )}
      title={`Run ${runId}`}
    >
      <Rabbit className="size-3" />
      {label}
    </Link>
  );
}
