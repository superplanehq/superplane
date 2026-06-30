import type { CanvasesCanvasVersion } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { cn } from "@/lib/utils";
import { formatCommitMessage, formatCommitSha } from "@/pages/app/lib/canvas-versions";
import { useCallback } from "react";
import type { KeyboardEvent as ReactKeyboardEvent } from "react";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";

export function VersionRow({
  version,
  isActive = false,
  isBranchHead = false,
  rowTestId,
  onUseVersion,
}: {
  version: CanvasesCanvasVersion;
  isActive?: boolean;
  isBranchHead?: boolean;
  rowTestId?: string;
  onUseVersion: (versionID: string) => void;
}) {
  const { versionID, commitMessage, commitSha, timestamp } = deriveCommitRowFields(version);

  const handleRowActivate = useCallback(() => {
    onUseVersion(versionID);
  }, [onUseVersion, versionID]);

  const handleRowKeyDown = useCallback(
    (event: ReactKeyboardEvent<HTMLDivElement>) => {
      if (!isActivationKey(event.key)) return;
      event.preventDefault();
      onUseVersion(versionID);
    },
    [onUseVersion, versionID],
  );

  if (!versionID) {
    return null;
  }

  const ariaLabel = commitSha ? `${commitSha} ${commitMessage}` : commitMessage;

  return (
    <div
      data-testid={rowTestId}
      className={versionRowClassName(isActive)}
      role="button"
      tabIndex={0}
      onClick={handleRowActivate}
      onKeyDown={handleRowKeyDown}
      aria-label={ariaLabel}
      title={ariaLabel}
    >
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        {commitSha ? <span className="font-mono text-[11px] font-medium text-slate-600">{commitSha}</span> : null}
        <span
          className={cn(
            "min-w-0 truncate text-xs",
            isActive ? "font-semibold text-sky-900" : "font-medium text-slate-900",
          )}
        >
          {commitMessage}
        </span>
      </div>
      {isBranchHead ? (
        <span className="shrink-0 rounded bg-sky-200 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-sky-800">
          HEAD
        </span>
      ) : null}
      {timestamp ? (
        <TimeAgo date={timestamp} includeAgo={false} className="shrink-0 text-xs tabular-nums text-slate-500" />
      ) : null}
    </div>
  );
}

function deriveCommitRowFields(version: CanvasesCanvasVersion) {
  return {
    versionID: version.metadata?.id ?? "",
    commitMessage: formatCommitMessage(version),
    commitSha: formatCommitSha(version),
    timestamp: version.metadata?.createdAt || version.metadata?.updatedAt,
  };
}

function isActivationKey(key: string): boolean {
  return key === "Enter" || key === " ";
}

function versionRowClassName(isActive: boolean): string {
  return cn(
    RUNS_SIDEBAR_ROW_CLASS,
    "group w-full cursor-pointer text-left transition-colors",
    isActive ? "bg-sky-100" : "bg-white hover:bg-gray-50",
  );
}
