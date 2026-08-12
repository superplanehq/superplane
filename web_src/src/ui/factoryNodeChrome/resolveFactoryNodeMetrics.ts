import React from "react";
import { Timestamp } from "@/components/Timestamp";
import { formatFactoryNodeDuration } from "./status";
import type { FactoryNodeStatus } from "./types";

export type FactoryNodeMetricsSection = {
  showAutomaticTime?: boolean;
  receivedAt?: Date;
  eventSubtitle?: string | React.ReactNode;
};

/**
 * Right-side footer value for factory cards — same sources as classic
 * EventSectionDisplay: live duration, eventSubtitle (often a TimeAgo node),
 * or Timestamp from receivedAt.
 */
export function resolveFactoryNodeMetrics({
  status,
  section,
  liveDurationMs,
}: {
  status: FactoryNodeStatus;
  section: FactoryNodeMetricsSection | undefined;
  liveDurationMs: number | null;
}): React.ReactNode | null {
  if (!section) return null;

  const isLiveStatus = status === "running" || status === "cancelling";
  const liveText =
    isLiveStatus && liveDurationMs !== null ? formatFactoryNodeDuration(liveDurationMs, { soFar: true }) : null;

  if (section.eventSubtitle) {
    if (section.showAutomaticTime && liveText) {
      return liveText;
    }
    if (typeof section.eventSubtitle === "string") {
      const trimmed = section.eventSubtitle.trim();
      return trimmed.length > 0 ? trimmed : null;
    }
    return section.eventSubtitle;
  }

  if (liveText) {
    return liveText;
  }

  if (section.receivedAt) {
    return React.createElement(Timestamp, {
      date: section.receivedAt,
      display: "relative",
      relativeStyle: "abbreviated",
    });
  }

  return null;
}
