import React from "react";
import type { DraftDiffStatus } from "@/lib/draftDiff";
import type { MetadataItem } from "../metadataList";
import { FactoryNodeCardShell } from "./FactoryNodeCardShell";
import { NodeHoverActions } from "./NodeHoverActions";
import { WarningBadge } from "./WarningBadge";
import { formatFactoryNodeDuration, normalizeFactoryNodeStatus } from "./status";
import type { FactoryNodeStatus } from "./types";

/** Minimal event slice — avoids importing ComponentBase (circular). */
export type FactoryNodeEventSlice = {
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
};

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

function useFactoryNodeMetrics(status: FactoryNodeStatus, section: FactoryNodeEventSlice | undefined): string | null {
  const [liveDuration, setLiveDuration] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (status !== "running" || !section?.receivedAt) {
      setLiveDuration(null);
      return;
    }
    const receivedAt = section.receivedAt;
    setLiveDuration(Date.now() - receivedAt.getTime());
    const interval = setInterval(() => {
      setLiveDuration(Date.now() - receivedAt.getTime());
    }, 1000);
    return () => clearInterval(interval);
  }, [status, section?.receivedAt]);

  if (status === "running" && liveDuration !== null) {
    return formatFactoryNodeDuration(liveDuration, { soFar: true });
  }
  if (status === "pending") {
    return null;
  }
  if (typeof section?.eventSubtitle === "string" && section.eventSubtitle.trim()) {
    return section.eventSubtitle.trim();
  }
  return null;
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
}: FactoryNodeCardProps) {
  const primarySection = eventSections?.[0];
  const status = normalizeFactoryNodeStatus(primarySection?.eventState);
  const metrics = useFactoryNodeMetrics(status, primarySection);
  const subtitle = resolveSubtitle(metadata, eventSections);
  const badgeText = error?.trim() || warning?.trim() || "";

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
      />
    </div>
  );
}
