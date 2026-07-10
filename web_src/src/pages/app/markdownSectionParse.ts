import type { ReactNode } from "react";

import { splitBlockquoteMarkerLine } from "./markdownBlockquoteMarker";

const SECTION_MARKER_RE = /^\[!SECTION(?::([a-z0-9_-]+))?\]\s+(.+)$/i;

export type ParsedMarkdownSection = {
  title: string;
  trailing?: string;
  presetId?: string;
  body: ReactNode;
};

/**
 * If `children` are a `[!SECTION] Title` or `[!SECTION:preset] Title` blockquote,
 * return the title/preset/body with the marker line removed.
 *
 * Optional trailing meta after a middle-dot separator:
 * `> [!SECTION:runbook] Deploy checklist · prod`
 */
export function parseGithubSectionChildren(children: ReactNode): ParsedMarkdownSection | null {
  const split = splitBlockquoteMarkerLine(children);
  if (!split) {
    return null;
  }

  const match = split.markerText.match(SECTION_MARKER_RE);
  if (!match) {
    return null;
  }

  const presetId = match[1]?.toLowerCase();
  const rawTitle = match[2].trim();
  if (!rawTitle) {
    return null;
  }

  const { title, trailing } = splitSectionTitle(rawTitle);
  return { title, trailing, presetId, body: split.body };
}

function splitSectionTitle(rawTitle: string): { title: string; trailing?: string } {
  const separator = " · ";
  const separatorIndex = rawTitle.indexOf(separator);
  if (separatorIndex < 0) {
    return { title: rawTitle };
  }

  const title = rawTitle.slice(0, separatorIndex).trim();
  const trailing = rawTitle.slice(separatorIndex + separator.length).trim();
  if (!title) {
    return { title: rawTitle };
  }

  return trailing ? { title, trailing } : { title };
}
