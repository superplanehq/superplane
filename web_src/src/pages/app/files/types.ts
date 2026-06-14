import type { CSSProperties } from "react";

export type AppFile = {
  path: string;
  content: string;
  language?: string;
  loading?: boolean;
  errorMessage?: string;
};

// Diff of a single file's effective draft content against the live (main)
// version, mirroring the canvas tab's "draft vs live" comparison. liveContent is
// the baseline (live/main); draftContent is the effective draft (committed draft
// plus uncommitted staging, or the in-session pending edit when present).
export type FileDiffVersusLive = {
  path: string;
  liveContent: string;
  draftContent: string;
};

export type PendingFileChange =
  | {
      type: "added";
      path: string;
      content: string;
    }
  | {
      type: "modified";
      path: string;
      content: string;
    }
  | {
      type: "deleted";
      path: string;
    };

export const repositoryFileTreeStyle = {
  height: "100%",
  colorScheme: "light",
  "--trees-bg-override": "#ffffff",
  "--trees-bg-muted-override": "#f1f5f9",
  "--trees-border-color-override": "rgba(15, 23, 42, 0.15)",
  "--trees-fg-override": "#334155",
  "--trees-fg-muted-override": "#64748b",
  "--trees-focus-ring-color-override": "#0f172a",
  "--trees-selected-bg-override": "#e0f2fe",
  "--trees-selected-fg-override": "#020617",
  "--trees-padding-inline-override": "0px",
  "--trees-item-margin-x-override": "0px",
  "--trees-border-radius-override": "0px",
  "--trees-scrollbar-gutter-override": "0px",
  "--trees-action-lane-width-override": "0px",
} as CSSProperties;
