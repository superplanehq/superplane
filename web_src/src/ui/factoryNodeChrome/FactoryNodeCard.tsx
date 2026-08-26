import React from "react";
import type { DraftDiffStatus } from "@/lib/draftDiff";
import type { EventStateMap } from "../componentBase/eventState";
import type { MetadataItem } from "../metadataList";
import { FactoryNodeCardShell } from "./FactoryNodeCardShell";
import { NodeHoverActions } from "./NodeHoverActions";
import { WarningBadge } from "./WarningBadge";
import { resolveFactoryNodeCardTitles } from "./resolveFactoryNodeCardTitles";
import { resolveFactoryNodeMetrics } from "./resolveFactoryNodeMetrics";
import { shouldShowFactoryNodeStatusFooter } from "./shouldShowFactoryNodeStatusFooter";
import { resolveFactoryRuntimeStatus } from "./status";
import type { FactoryNodeStatus } from "./types";

/** Minimal event slice — avoids importing ComponentBase (circular). */
export type FactoryNodeEventSlice = {
  showAutomaticTime?: boolean;
  receivedAt?: Date;
  eventState?: string;
  eventTitle?: string;
  eventSubtitle?: string | React.ReactNode;
};

export type FactoryNodeCardProps = {
  title: string;
  /** Go Action.Label() — primary title on factory cards. */
  componentLabel?: string;
  /** User-given node name — gray subtitle on factory cards. */
  nodeName?: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  selected?: boolean;
  metadata?: MetadataItem[];
  eventSections?: FactoryNodeEventSlice[];
  error?: string;
  warning?: string;
  draftDiffStatus?: DraftDiffStatus;
  dimBodyBelowHeader?: boolean;
  showHeader?: boolean;
  onDuplicate?: () => void;
  onDelete?: () => void;
  onToggleView?: () => void;
  isCompactView?: boolean;
  /** Edit mode shows a neutral "No run" footer instead of runtime status. */
  canvasMode?: "live" | "edit";
  /** False on factory Live without a selected run (topology only). */
  showRuntimeStatus?: boolean;
  /**
   * Run inspection: true while the selected run can still progress.
   * False when finished — empty nodes become Did not run instead of Pending.
   * Defaults true when unknown.
   */
  runIsActive?: boolean;
  /** Mapper eventStateMap — same source the run sidebar uses for badge color. */
  eventStateMap?: EventStateMap;
  /** Optional content between the node header and status footer. */
  body?: React.ReactNode;
};

function useFactoryNodeMetrics(
  status: FactoryNodeStatus,
  section: FactoryNodeEventSlice | undefined,
): React.ReactNode | null {
  const [liveDuration, setLiveDuration] = React.useState<number | null>(null);

  const isLiveStatus = status === "running" || status === "cancelling";

  React.useEffect(() => {
    if (!isLiveStatus || !section?.receivedAt) {
      setLiveDuration(null);
      return;
    }
    const receivedAt = section.receivedAt;
    setLiveDuration(Date.now() - receivedAt.getTime());
    const interval = setInterval(() => {
      setLiveDuration(Date.now() - receivedAt.getTime());
    }, 1000);
    return () => clearInterval(interval);
  }, [isLiveStatus, section?.receivedAt]);

  return resolveFactoryNodeMetrics({
    status,
    section,
    liveDurationMs: liveDuration,
  });
}

function resolveFactoryCardFooter({
  canvasMode,
  isCompactView,
  showRuntimeStatus,
  runtimeStatus,
  runtimeMetrics,
}: {
  canvasMode: "live" | "edit";
  isCompactView?: boolean;
  showRuntimeStatus: boolean;
  runtimeStatus: FactoryNodeStatus;
  runtimeMetrics: React.ReactNode | null;
}): {
  status: FactoryNodeStatus;
  metrics: React.ReactNode | null;
  statusLabel: string | undefined;
  showStatusFooter: boolean;
} {
  const isEditMode = canvasMode === "edit";
  return {
    status: isEditMode ? "pending" : runtimeStatus,
    metrics: isEditMode ? null : runtimeMetrics,
    statusLabel: isEditMode ? "No run" : undefined,
    showStatusFooter: shouldShowFactoryNodeStatusFooter({
      canvasMode,
      isCompactView,
      showRuntimeStatus,
    }),
  };
}

/**
 * Factory-app node chrome: white/token card + provider icon + tinted status footer.
 * Used only when `isFactoryApp` is set on ComponentBase (vertical factory canvases).
 */
export function FactoryNodeCard({
  title,
  componentLabel,
  nodeName,
  iconSrc,
  iconSlug,
  iconColor,
  selected = false,
  eventSections,
  error,
  warning,
  draftDiffStatus,
  dimBodyBelowHeader = false,
  showHeader = true,
  onDuplicate,
  onDelete,
  onToggleView,
  isCompactView,
  canvasMode = "live",
  showRuntimeStatus = true,
  runIsActive = true,
  eventStateMap,
  body,
}: FactoryNodeCardProps) {
  const primarySection = eventSections?.[0];
  const runtimeStatus = resolveFactoryRuntimeStatus({
    eventState: primarySection?.eventState,
    runIsActive,
    stateMap: eventStateMap,
  });
  const runtimeMetrics = useFactoryNodeMetrics(runtimeStatus, primarySection);
  const { title: cardTitle, subtitle } = resolveFactoryNodeCardTitles({
    componentLabel,
    nodeName,
    fallbackTitle: title,
  });
  const badgeText = error?.trim() || warning?.trim() || "";
  const footer = resolveFactoryCardFooter({
    canvasMode,
    isCompactView,
    showRuntimeStatus,
    runtimeStatus,
    runtimeMetrics,
  });

  return (
    <div className="group relative" data-view-mode={isCompactView ? "compact" : "expanded"}>
      <NodeHoverActions
        showHeader={showHeader}
        onDuplicate={onDuplicate}
        onDelete={onDelete}
        onToggleView={onToggleView}
        isCompactView={isCompactView}
      />
      {badgeText ? <WarningBadge text={badgeText} /> : null}
      <FactoryNodeCardShell
        title={cardTitle}
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        iconColor={iconColor}
        selected={selected}
        subtitle={subtitle}
        status={footer.status}
        metrics={footer.metrics}
        statusLabel={footer.statusLabel}
        draftDiffStatus={draftDiffStatus}
        dimBodyBelowHeader={dimBodyBelowHeader}
        isCompactView={isCompactView}
        showStatusFooter={footer.showStatusFooter}
        body={body}
      />
    </div>
  );
}
