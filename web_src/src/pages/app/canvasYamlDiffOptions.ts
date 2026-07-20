import type { ResolvedTheme } from "@/lib/themePreference";
import { DARK_BASE_BG_HEX } from "@/lib/darkThemeSurfaces";

const LIGHT_DIFF_CSS = `
  :host {
    display: block;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    font-size: 12px;
    line-height: 18px;
    --diffs-bg-context-override: #ffffff;
    --diffs-bg-context-gutter-override: #ffffff;
    --diffs-bg-deletion-override: #fee2e2;
    --diffs-bg-addition-override: #dcfce7;
    --diffs-bg-deletion-emphasis-override: #fecaca;
    --diffs-bg-addition-emphasis-override: #bbf7d0;
  }

  [data-line-type="context"],
  [data-line-type="context-expanded"] {
    --diffs-line-bg: #ffffff;
    --diffs-computed-diff-line-bg: #ffffff;
    --diffs-computed-selected-line-bg: #ffffff;
  }

  [data-diffs-header] {
    border-bottom: 1px solid rgb(226 232 240);
    background: rgb(255 255 255);
    z-index: 5;
  }

  [data-diffs-header="custom"] {
    display: block;
    padding: 0;
  }
`;

const DARK_DIFF_CSS = `
  :host {
    display: block;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    font-size: 12px;
    line-height: 18px;
    --diffs-bg-context-override: ${DARK_BASE_BG_HEX};
    --diffs-bg-context-gutter-override: ${DARK_BASE_BG_HEX};
    --diffs-bg-deletion-override: #3f1d1d;
    --diffs-bg-addition-override: #14311f;
    --diffs-bg-deletion-emphasis-override: #5b2626;
    --diffs-bg-addition-emphasis-override: #1c4a2c;
  }

  [data-line-type="context"],
  [data-line-type="context-expanded"] {
    --diffs-line-bg: ${DARK_BASE_BG_HEX};
    --diffs-computed-diff-line-bg: ${DARK_BASE_BG_HEX};
    --diffs-computed-selected-line-bg: ${DARK_BASE_BG_HEX};
  }

  [data-diffs-header] {
    border-bottom: 1px solid rgb(55 65 81);
    background: ${DARK_BASE_BG_HEX};
    z-index: 5;
  }

  [data-diffs-header="custom"] {
    display: block;
    padding: 0;
  }
`;

export const getCanvasYamlDiffOptions = (resolvedTheme: ResolvedTheme) =>
  ({
    theme: resolvedTheme === "dark" ? "github-dark" : "github-light",
    themeType: resolvedTheme === "dark" ? "dark" : "light",
    diffStyle: "split",
    diffIndicators: "classic",
    hunkSeparators: "line-info",
    lineDiffType: "word",
    overflow: "wrap",
    stickyHeader: true,
    tokenizeMaxLineLength: 1_000,
    parseDiffOptions: { context: 6 },
    unsafeCSS: resolvedTheme === "dark" ? DARK_DIFF_CSS : LIGHT_DIFF_CSS,
  }) as const;
