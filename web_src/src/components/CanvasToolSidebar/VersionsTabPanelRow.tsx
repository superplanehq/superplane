import type { CanvasesCanvasVersion } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { cn } from "@/lib/utils";
import { useCallback } from "react";
import type { KeyboardEvent as ReactKeyboardEvent } from "react";
import { formatVersionLabel } from "@/pages/app/lib/canvas-versions";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";

export function VersionRow({
  version,
  isActive = false,
  isCurrentLive = false,
  isFirstCanvasVersion = false,
  rowTestId,
  onUseVersion,
}: {
  version: CanvasesCanvasVersion;
  isActive?: boolean;
  isCurrentLive?: boolean;
  isFirstCanvasVersion?: boolean;
  rowTestId?: string;
  onUseVersion: (versionID: string) => void;
}) {
  const { versionID, ownerName, versionLabel, timestamp } = deriveVersionRowFields(version, isFirstCanvasVersion);

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

  return (
    <div
      data-testid={rowTestId}
      className={versionRowClassName(isActive)}
      role="button"
      tabIndex={0}
      onClick={handleRowActivate}
      onKeyDown={handleRowKeyDown}
      aria-label={`Preview ${versionLabel}`}
      title={`${versionLabel} · ${ownerName}`}
    >
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-xs",
          isActive ? "font-semibold text-sky-900" : "font-medium text-slate-900",
        )}
      >
        {versionLabel}
      </span>
      {isCurrentLive ? (
        <span className="shrink-0 rounded bg-sky-200 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-sky-800">
          Current
        </span>
      ) : null}
      <span className="max-w-[40%] shrink-0 truncate text-[11px] text-slate-500">{ownerName}</span>
      {timestamp ? (
        <TimeAgo date={timestamp} includeAgo={false} className="shrink-0 text-xs tabular-nums text-slate-500" />
      ) : null}
    </div>
  );
}

function deriveVersionRowFields(version: CanvasesCanvasVersion, isFirstCanvasVersion: boolean) {
  return {
    versionID: version.metadata?.id ?? "",
    ownerName: version.metadata?.owner?.name || "Unknown owner",
    versionLabel: isFirstCanvasVersion ? "v1" : formatVersionLabel(version),
    timestamp: version.metadata?.updatedAt || version.metadata?.createdAt,
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
