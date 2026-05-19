import type { CanvasChangeManagement, CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Ellipsis } from "lucide-react";
import { useCallback } from "react";
import type { KeyboardEvent as ReactKeyboardEvent, MouseEvent as ReactMouseEvent } from "react";
import { getChangeRequestReviewPhase } from "@/pages/workflowv2/changeRequestReviewActions";
import { formatVersionLabel, formatVersionTimestamp } from "@/pages/workflowv2/lib/canvas-versions";

type ActiveReviewPhase = Exclude<ReturnType<typeof getChangeRequestReviewPhase>, { kind: "none" }>;

export function VersionRow({
  version,
  changeRequest,
  changeRequestApprovalConfig,
  previousVersion,
  variant = "default",
  isActive = false,
  isCurrentLive = false,
  isFirstCanvasVersion = false,
  rowTestId,
  onUseVersion,
  onViewDiff,
}: {
  version: CanvasesCanvasVersion;
  changeRequest?: CanvasesCanvasChangeRequest;
  changeRequestApprovalConfig?: CanvasChangeManagement;
  previousVersion?: CanvasesCanvasVersion;
  variant?: "default" | "rejected";
  isActive?: boolean;
  isCurrentLive?: boolean;
  isFirstCanvasVersion?: boolean;
  rowTestId?: string;
  onUseVersion: (versionID: string) => void;
  onViewDiff: (
    version: CanvasesCanvasVersion,
    previousVersion: CanvasesCanvasVersion,
    changeRequest?: CanvasesCanvasChangeRequest,
  ) => void;
}) {
  const versionID = version.metadata?.id ?? "";

  const viewModel = buildVersionRowViewModel({
    version,
    changeRequest,
    changeRequestApprovalConfig,
    variant,
    isCurrentLive,
    isFirstCanvasVersion,
  });

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
      className={versionRowClassName({
        isActive,
        isCurrentLive,
        variant,
        activeReviewPhase: viewModel.activeReviewPhase,
      })}
      role="button"
      tabIndex={0}
      onClick={handleRowActivate}
      onKeyDown={handleRowKeyDown}
      aria-label={`Preview ${viewModel.versionLabel}`}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-slate-900">{viewModel.versionLabel}</p>
          <VersionSubtitle
            isCurrentLive={isCurrentLive}
            variant={variant}
            activeReviewPhase={viewModel.activeReviewPhase}
            versionSubtitle={viewModel.versionSubtitle}
          />
        </div>
        <VersionDetailsButton
          version={version}
          previousVersion={previousVersion}
          changeRequest={changeRequest}
          onViewDiff={onViewDiff}
        />
      </div>
    </div>
  );
}

function buildVersionRowViewModel({
  version,
  changeRequest,
  changeRequestApprovalConfig,
  variant,
  isCurrentLive,
  isFirstCanvasVersion,
}: {
  version: CanvasesCanvasVersion;
  changeRequest?: CanvasesCanvasChangeRequest;
  changeRequestApprovalConfig?: CanvasChangeManagement;
  variant: "default" | "rejected";
  isCurrentLive: boolean;
  isFirstCanvasVersion: boolean;
}) {
  const ownerName = version.metadata?.owner?.name || "Unknown owner";
  const changeRequestTitle = changeRequest?.metadata?.title?.trim();
  const reviewPhase = getChangeRequestReviewPhase(changeRequest, changeRequestApprovalConfig);
  return {
    versionLabel: isFirstCanvasVersion ? "v1" : changeRequestTitle || formatVersionLabel(version),
    versionSubtitle: formatVersionSubtitle(version, ownerName),
    activeReviewPhase: resolveActiveReviewPhase(reviewPhase, variant, isCurrentLive),
  };
}

function formatVersionSubtitle(version: CanvasesCanvasVersion, ownerName: string): string {
  const versionTimestamp = formatVersionTimestamp(version);
  return versionTimestamp ? `${ownerName} on ${versionTimestamp}` : ownerName;
}

function resolveActiveReviewPhase(
  reviewPhase: ReturnType<typeof getChangeRequestReviewPhase>,
  variant: "default" | "rejected",
  isCurrentLive: boolean,
): ActiveReviewPhase | null {
  if (variant === "rejected" || isCurrentLive || reviewPhase.kind === "none") {
    return null;
  }
  return reviewPhase;
}

function isActivationKey(key: string): boolean {
  return key === "Enter" || key === " ";
}

function versionRowClassName({
  isActive,
  isCurrentLive,
  variant,
  activeReviewPhase,
}: {
  isActive: boolean;
  isCurrentLive: boolean;
  variant: "default" | "rejected";
  activeReviewPhase: ActiveReviewPhase | null;
}): string {
  const baseClassName = "w-full cursor-pointer rounded-md px-2.5 py-2 text-left transition";
  if (!isActive) {
    return `${baseClassName} bg-white hover:bg-slate-100`;
  }
  if (isCurrentLive) {
    return `${baseClassName} bg-green-100`;
  }
  if (variant === "rejected") {
    return `${baseClassName} bg-red-50`;
  }
  return cn(baseClassName, activeReviewPhase ? activeReviewPhase.sidebarRowActiveClassName : "bg-sky-100");
}

function VersionSubtitle({
  isCurrentLive,
  variant,
  activeReviewPhase,
  versionSubtitle,
}: {
  isCurrentLive: boolean;
  variant: "default" | "rejected";
  activeReviewPhase: ActiveReviewPhase | null;
  versionSubtitle: string;
}) {
  const statusIndicator = buildStatusIndicator(isCurrentLive, variant, activeReviewPhase);

  return (
    <p className="mt-0.5 truncate text-xs text-slate-600">
      {statusIndicator ? (
        <>
          {statusIndicator.dotClassName ? <span className={cn("mr-1", statusIndicator.dotClassName)}>●</span> : null}
          <StatusBadge className={statusIndicator.labelClassName} label={statusIndicator.label} />
        </>
      ) : null}
      {versionSubtitle}
    </p>
  );
}

function buildStatusIndicator(
  isCurrentLive: boolean,
  variant: "default" | "rejected",
  activeReviewPhase: ActiveReviewPhase | null,
) {
  if (isCurrentLive) {
    return { label: "Live", labelClassName: "font-medium text-green-600" };
  }
  if (variant === "rejected") {
    return { label: "Rejected", labelClassName: "font-medium text-red-600" };
  }
  if (!activeReviewPhase) {
    return null;
  }
  return {
    label: activeReviewPhase.label,
    labelClassName: activeReviewPhase.labelClassName,
    dotClassName: activeReviewPhase.dotClassName,
  };
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
  changeRequest,
  onViewDiff,
}: {
  version: CanvasesCanvasVersion;
  previousVersion?: CanvasesCanvasVersion;
  changeRequest?: CanvasesCanvasChangeRequest;
  onViewDiff: (
    version: CanvasesCanvasVersion,
    previousVersion: CanvasesCanvasVersion,
    changeRequest?: CanvasesCanvasChangeRequest,
  ) => void;
}) {
  const handleClick = useCallback(
    (event: ReactMouseEvent<HTMLButtonElement>) => {
      event.stopPropagation();
      if (!previousVersion) return;
      onViewDiff(version, previousVersion, changeRequest);
    },
    [changeRequest, onViewDiff, previousVersion, version],
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
              aria-label="View details"
            >
              <Ellipsis className="h-4 w-4" />
            </Button>
          </span>
        </TooltipTrigger>
        <TooltipContent side="top">View details</TooltipContent>
      </Tooltip>
    </div>
  );
}
