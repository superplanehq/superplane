import { useState } from "react";
import { CopyButton } from "@/ui/CopyButton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { RunStatus } from "./types";

const statusColors: Record<RunStatus, string> = {
  Running: "bg-blue-500",
  Passed: "bg-emerald-500",
  Failed: "bg-red-400",
  Cancelled: "bg-slate-400",
};

export function RunStatusLabel({ status }: { status: RunStatus }) {
  return (
    <span className="inline-flex items-center gap-2 whitespace-nowrap text-sm">
      <span aria-hidden className={`size-2 rounded-full ${statusColors[status]}`} />
      {status}
    </span>
  );
}

export function RunIdCopy({ id, full = false }: { id: string; full?: boolean }) {
  const [copyError, setCopyError] = useState(false);
  return (
    <div className="flex flex-wrap items-center gap-2">
      {full && <code className="break-all text-xs text-muted-foreground">{id}</code>}
      <Tooltip>
        <TooltipTrigger asChild>
          <span tabIndex={0}>
            <CopyButton
              text={id}
              variant={full ? "button" : "icon"}
              ariaLabel={`Copy full run ID ${id}`}
              copiedAriaLabel="Run ID copied"
              copiedLabel="Run ID copied"
              onCopyError={() => setCopyError(true)}
            >
              {full ? "Copy full ID" : undefined}
            </CopyButton>
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <code>{id}</code>
          <div>Copy full run ID</div>
        </TooltipContent>
      </Tooltip>
      {copyError && (
        <span role="alert" className="text-xs text-red-500">
          Could not copy. Select the full ID: {id}
        </span>
      )}
    </div>
  );
}
