import type { CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Diff } from "lucide-react";
import { useCallback } from "react";
import type { KeyboardEvent as ReactKeyboardEvent, MouseEvent as ReactMouseEvent } from "react";
import { formatVersionLabel, formatVersionTimestamp } from "@/pages/app/lib/canvas-versions";

export function VersionRow({
  version,
  previousVersion,
  isActive = false,
  isCurrentLive = false,
  isFirstCanvasVersion = false,
  rowTestId,
  onUseVersion,
  onViewDiff,
}: {
  version: CanvasesCanvasVersion;
  previousVersion?: CanvasesCanvasVersion;
  isActive?: boolean;
  isCurrentLive?: boolean;
  isFirstCanvasVersion?: boolean;
  rowTestId?: string;
  onUseVersion: (versionID: string) => void;
  onViewDiff: (version: CanvasesCanvasVersion, previousVersion: CanvasesCanvasVersion) => void;
}) {
  const versionID = version.metadata?.id ?? "";
  const ownerName = version.metadata?.owner?.name || "Unknown owner";
  const versionLabel = isFirstCanvasVersion ? "v1" : formatVersionTimestamp(version) || formatVersionLabel(version);

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
    >
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p className="truncate text-[13px] font-medium text-slate-900">{versionLabel}</p>
          <VersionSubtitle isCurrentLive={isCurrentLive} ownerName={ownerName} />
        </div>
        <VersionDetailsButton version={version} previousVersion={previousVersion} onViewDiff={onViewDiff} />
      </div>
    </div>
  );
}

function isActivationKey(key: string): boolean {
  return key === "Enter" || key === " ";
}

function versionRowClassName(isActive: boolean): string {
  const baseClassName = "w-full cursor-pointer border-b border-b-slate-950/10 px-4 py-2 text-left transition";
  if (!isActive) {
    return `${baseClassName} bg-white hover:bg-slate-100`;
  }
  return cn(baseClassName, "bg-sky-100");
}

function VersionSubtitle({ isCurrentLive, ownerName }: { isCurrentLive: boolean; ownerName: string }) {
  return (
    <p className="mt-0.5 truncate text-xs text-slate-600">
      {isCurrentLive ? <StatusBadge className="font-medium text-sky-700" label="Current Version" /> : null}
      {ownerName}
    </p>
  );
}

function StatusBadge({ className, label }: { className: string; label: string }) {
  return (
    <>
      <span className={className}>{label}</span> {"·"}{" "}
    </>
  );
}

function VersionDetailsButton({
  version,
  previousVersion,
  onViewDiff,
}: {
  version: CanvasesCanvasVersion;
  previousVersion?: CanvasesCanvasVersion;
  onViewDiff: (version: CanvasesCanvasVersion, previousVersion: CanvasesCanvasVersion) => void;
}) {
  const handleClick = useCallback(
    (event: ReactMouseEvent<HTMLButtonElement>) => {
      event.stopPropagation();
      if (!previousVersion) return;
      onViewDiff(version, previousVersion);
    },
    [onViewDiff, previousVersion, version],
  );

  if (!previousVersion) return null;

  return (
    <div className="flex items-center gap-1.5">
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className="h-7 w-7 hover:bg-black/5 dark:hover:bg-black/5"
              onClick={handleClick}
              aria-label="View Diff"
            >
              <Diff className="size-3.5" />
            </Button>
          </span>
        </TooltipTrigger>
        <TooltipContent side="top">View Diff</TooltipContent>
      </Tooltip>
    </div>
  );
}
