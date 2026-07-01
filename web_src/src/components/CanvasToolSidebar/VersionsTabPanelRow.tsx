import type { CanvasesCanvasVersion } from "@/api-client";
import { Avatar } from "@/components/Avatar/avatar";
import { TimeAgo } from "@/components/TimeAgo";
import type { UserDisplayProfile } from "@/lib/userRefDisplay";
import { cn } from "@/lib/utils";
import { formatCommitMessage, formatCommitSha } from "@/pages/app/lib/canvas-versions";
import { useCallback } from "react";
import type { KeyboardEvent as ReactKeyboardEvent } from "react";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";

export function VersionRow({
  version,
  isActive = false,
  rowTestId,
  committer,
  onUseVersion,
}: {
  version: CanvasesCanvasVersion;
  isActive?: boolean;
  rowTestId?: string;
  committer?: UserDisplayProfile;
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
      {committer ? (
        <Avatar
          src={committer.avatarUrl}
          initials={committer.initials}
          alt={committer.name}
          className="size-6 shrink-0 bg-slate-700 text-slate-100"
          data-testid="committer-avatar"
        />
      ) : null}
      <div className="min-w-0 flex-1 truncate">
        <span
          className={cn(
            "block truncate text-xs",
            isActive ? "font-semibold text-sky-900" : "font-medium text-slate-900",
          )}
        >
          {commitMessage}
        </span>
      </div>
      {timestamp || commitSha ? (
        <div className="flex shrink-0 items-center gap-1.5 text-xs tabular-nums text-slate-500">
          {commitSha ? (
            <span className="shrink-0 font-mono text-[11px] font-medium text-slate-600">{commitSha}</span>
          ) : null}
          {commitSha && timestamp ? <span className="text-slate-300">·</span> : null}
          {timestamp ? <TimeAgo date={timestamp} includeAgo={false} className="shrink-0" /> : null}
        </div>
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
