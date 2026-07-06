const CANVAS_DIFF_UNSAFE_CSS = `
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

export const CANVAS_YAML_DIFF_OPTIONS = {
  theme: "github-light",
  diffStyle: "split",
  diffIndicators: "classic",
  hunkSeparators: "line-info",
  lineDiffType: "word",
  overflow: "wrap",
  stickyHeader: true,
  tokenizeMaxLineLength: 1_000,
  parseDiffOptions: { context: 6 },
  unsafeCSS: CANVAS_DIFF_UNSAFE_CSS,
} as const;
