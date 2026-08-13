import React from "react";
import type { DraftDiffStatus } from "@/lib/draftDiff";
import type { MetadataItem } from "../metadataList";
import { FactoryNodeCardShell } from "./FactoryNodeCardShell";
import { NodeHoverActions } from "./NodeHoverActions";
import { WarningBadge } from "./WarningBadge";
import { resolveFactoryNodeCardTitles } from "./resolveFactoryNodeCardTitles";
import { resolveFactoryNodeMetrics } from "./resolveFactoryNodeMetrics";
import { shouldShowFactoryNodeStatusFooter } from "./shouldShowFactoryNodeStatusFooter";
import { normalizeFactoryNodeStatus } from "./status";
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
}: FactoryNodeCardProps) {
  const primarySection = eventSections?.[0];
  const runtimeStatus = normalizeFactoryNodeStatus(primarySection?.eventState);
  const runtimeMetrics = useFactoryNodeMetrics(runtimeStatus, primarySection);
  const { title: cardTitle, subtitle } = resolveFactoryNodeCardTitles({
    componentLabel,
    nodeName,
    fallbackTitle: title,
  });
  const badgeText = error?.trim() || warning?.trim() || "";
  const isEditMode = canvasMode === "edit";
  const showStatusFooter = shouldShowFactoryNodeStatusFooter({
    canvasMode,
    isCompactView,
    showRuntimeStatus,
  });
  const status: FactoryNodeStatus = isEditMode ? "pending" : runtimeStatus;
  const metrics = isEditMode ? null : runtimeMetrics;
  const statusLabel = isEditMode ? "No run" : undefined;

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
        status={status}
        metrics={metrics}
        statusLabel={statusLabel}
        draftDiffStatus={draftDiffStatus}
        dimBodyBelowHeader={dimBodyBelowHeader}
        isCompactView={isCompactView}
        showStatusFooter={showStatusFooter}
      />
    </div>
  );
}
