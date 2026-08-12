import React from "react";
import type { DraftDiffStatus } from "@/lib/draftDiff";
import type { MetadataItem } from "../metadataList";
import { FactoryNodeCardShell } from "./FactoryNodeCardShell";
import { NodeHoverActions } from "./NodeHoverActions";
import { WarningBadge } from "./WarningBadge";
import { resolveFactoryNodeMetrics } from "./resolveFactoryNodeMetrics";
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
  /** Edit mode hides the runtime status footer (no fake Pending). */
  canvasMode?: "live" | "edit";
};

/** Status footer is for live/run views only — not while editing the graph. */
export function shouldShowFactoryNodeStatusFooter({
  canvasMode,
  isCompactView,
}: {
  canvasMode?: "live" | "edit";
  isCompactView?: boolean;
}): boolean {
  if (isCompactView) return false;
  if (canvasMode === "edit") return false;
  return true;
}

function resolveSubtitle(
  metadata: MetadataItem[] | undefined,
  eventSections: FactoryNodeEventSlice[] | undefined,
): string | null {
  const firstMeta = metadata?.[0]?.label;
  if (typeof firstMeta === "string" && firstMeta.trim()) {
    return firstMeta.trim();
  }
  const eventTitle = eventSections?.[0]?.eventTitle;
  if (typeof eventTitle === "string" && eventTitle.trim()) {
    return eventTitle.trim();
  }
  return null;
}

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
  iconSrc,
  iconSlug,
  iconColor,
  selected = false,
  metadata,
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
}: FactoryNodeCardProps) {
  const primarySection = eventSections?.[0];
  const status = normalizeFactoryNodeStatus(primarySection?.eventState);
  const metrics = useFactoryNodeMetrics(status, primarySection);
  const subtitle = resolveSubtitle(metadata, eventSections);
  const badgeText = error?.trim() || warning?.trim() || "";
  const showStatusFooter = shouldShowFactoryNodeStatusFooter({ canvasMode, isCompactView });

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
        title={title}
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        iconColor={iconColor}
        selected={selected}
        subtitle={subtitle}
        status={status}
        metrics={metrics}
        draftDiffStatus={draftDiffStatus}
        dimBodyBelowHeader={dimBodyBelowHeader}
        isCompactView={isCompactView}
        showStatusFooter={showStatusFooter}
      />
    </div>
  );
}
