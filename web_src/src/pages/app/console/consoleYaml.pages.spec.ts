import { describe, expect, it } from "vitest";

import type { ConsolePage } from "@/hooks/useCanvasData";

import {
  consoleToYaml,
  DEFAULT_CONSOLE_PAGE_ID,
  DEFAULT_CONSOLE_PAGE_NAME,
  MAX_CONSOLE_PAGES,
  MAX_CONSOLE_PANELS_PER_PAGE,
  parseConsoleYaml,
  parseConsoleYamlLenient,
  validateConsolePagesDelta,
} from "./consoleYaml";

describe("consoleToYaml / parseConsoleYaml — multi-page shape", () => {
  it("exports 2+ pages via spec.pages and drops the legacy top-level keys", () => {
    const text = consoleToYaml({
      pages: [
        {
          id: "overview",
          name: "Overview",
          panels: [{ id: "intro", type: "markdown", content: { body: "hi" } }],
          layout: [{ i: "intro", x: 0, y: 0, w: 12, h: 6 }],
        },
        {
          id: "details",
          name: "Details",
          panels: [],
          layout: [],
        },
      ],
    });

    expect(text).toContain("pages:");
    // The legacy top-level keys must not appear alongside `pages:` to
    // avoid ambiguity for downstream parsers (and importers).
    expect(text).not.toMatch(/\nlayout:/);

    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.pages).toHaveLength(2);
    expect(result.data.spec.pages[0]!.id).toBe("overview");
    expect(result.data.spec.pages[1]!.id).toBe("details");
  });

  it("rejects a document that mixes legacy panels/layout with pages", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: p
      type: markdown
      content: {}
  pages:
    - id: overview
      panels: []
      layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/pages cannot be combined/);
  });

  it("rejects too many pages", () => {
    let raw = "apiVersion: v1\nkind: Console\nmetadata: {}\nspec:\n  pages:\n";
    for (let i = 0; i < MAX_CONSOLE_PAGES + 1; i += 1) {
      raw += `    - id: page-${i}\n      panels: []\n      layout: []\n`;
    }
    const result = parseConsoleYaml(raw);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/Too many pages/);
  });

  it("rejects duplicate page ids", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  pages:
    - id: dup
      panels: []
      layout: []
    - id: dup
      panels: []
      layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/Duplicate page id/);
  });

  it("grandfathers over-cap consoles on the read path but blocks them on save", () => {
    // Simulate an existing console with more panels than the current
    // per-page cap allows. The lenient parser (used on the render path)
    // must accept it so the console still displays; the strict parser
    // (used on import / save) must reject the same document so no new
    // commit pushes a page further past the cap.
    let raw = "apiVersion: v1\nkind: Console\nmetadata: {}\nspec:\n  panels:\n";
    for (let i = 0; i < MAX_CONSOLE_PANELS_PER_PAGE + 3; i += 1) {
      raw += `    - id: panel-${i}\n      type: markdown\n      content: {}\n`;
    }
    raw += "  layout: []\n";

    const strict = parseConsoleYaml(raw);
    expect(strict.ok).toBe(false);
    if (!strict.ok) expect(strict.error).toMatch(/Too many panels/);

    const lenient = parseConsoleYamlLenient(raw);
    expect(lenient.ok).toBe(true);
    if (!lenient.ok) throw new Error(lenient.error);
    expect(lenient.data.spec.pages).toHaveLength(1);
    expect(lenient.data.spec.pages[0]!.panels).toHaveLength(MAX_CONSOLE_PANELS_PER_PAGE + 3);
  });

  it("escapes to spec.pages when the sole page has a non-default name so renames survive export", () => {
    // Renaming the only tab must round-trip through YAML. The legacy
    // `panels`/`layout` shape has no place to store `name`, so a
    // renamed single page must fall into the multi-page shape.
    const text = consoleToYaml({
      pages: [
        {
          id: DEFAULT_CONSOLE_PAGE_ID,
          name: "Deploy dashboard",
          panels: [{ id: "intro", type: "markdown", content: {} }],
          layout: [{ i: "intro", x: 0, y: 0, w: 12, h: 6 }],
        },
      ],
    });

    expect(text).toContain("pages:");
    expect(text).toContain("name: Deploy dashboard");

    const parsed = parseConsoleYaml(text);
    expect(parsed.ok).toBe(true);
    if (!parsed.ok) throw new Error(parsed.error);
    expect(parsed.data.spec.pages).toHaveLength(1);
    expect(parsed.data.spec.pages[0]!.name).toBe("Deploy dashboard");
  });

  it("escapes to spec.pages when the sole page has a non-default id", () => {
    const text = consoleToYaml({
      pages: [
        {
          id: "custom",
          name: DEFAULT_CONSOLE_PAGE_NAME,
          panels: [],
          layout: [],
        },
      ],
    });

    expect(text).toContain("pages:");
    expect(text).toContain("id: custom");
  });

  it("keeps emitting the legacy shape for the default single page (no diff noise)", () => {
    const text = consoleToYaml({
      pages: [
        {
          id: DEFAULT_CONSOLE_PAGE_ID,
          name: DEFAULT_CONSOLE_PAGE_NAME,
          panels: [{ id: "intro", type: "markdown", content: {} }],
          layout: [{ i: "intro", x: 0, y: 0, w: 12, h: 6 }],
        },
      ],
    });

    expect(text).not.toContain("pages:");
    expect(text).toContain("panels:");
  });

  it("mirrors backend delta rules: rejects new over-cap authoring without baseline", () => {
    // Fresh canvas (no baseline). Over-cap pages must be refused.
    const overCapPage: ConsolePage = {
      id: "big",
      name: "Big",
      panels: Array.from({ length: MAX_CONSOLE_PANELS_PER_PAGE + 1 }, (_v, i) => ({
        id: `p${i}`,
        type: "markdown",
        content: {},
      })),
      layout: [],
    };
    const err = validateConsolePagesDelta([overCapPage], []);
    expect(err).toContain("Too many panels");
  });

  it("mirrors backend delta rules: grandfathers over-cap pages by exact id", () => {
    // Committed had 25 panels; new keeps 25. Allowed.
    const grandfathered: ConsolePage = {
      id: "big",
      name: "Big",
      panels: Array.from({ length: MAX_CONSOLE_PANELS_PER_PAGE + 5 }, (_v, i) => ({
        id: `p${i}`,
        type: "markdown",
        content: {},
      })),
      layout: [],
    };
    expect(validateConsolePagesDelta([grandfathered], [grandfathered])).toBeNull();

    const shrunk: ConsolePage = {
      ...grandfathered,
      panels: grandfathered.panels.slice(0, MAX_CONSOLE_PANELS_PER_PAGE + 1),
    };
    expect(validateConsolePagesDelta([shrunk], [grandfathered])).toBeNull();

    // Growth past the previous count is still rejected.
    const grown: ConsolePage = {
      ...grandfathered,
      panels: [...grandfathered.panels, { id: "p-extra", type: "markdown", content: {} }],
    };
    const err = validateConsolePagesDelta([grown], [grandfathered]);
    expect(err).toContain("Too many panels");
  });

  it("mirrors backend delta rules: grandfathers renames via positional + panel-id-subset", () => {
    const previous: ConsolePage = {
      id: "main",
      name: "Main",
      panels: Array.from({ length: MAX_CONSOLE_PANELS_PER_PAGE + 3 }, (_v, i) => ({
        id: `p${i}`,
        type: "markdown",
        content: {},
      })),
      layout: [],
    };
    // Renamed id, same position, panels are a subset of the previous ids.
    const renamed: ConsolePage = { ...previous, id: "overview", name: "Overview" };
    expect(validateConsolePagesDelta([renamed], [previous])).toBeNull();

    // A "fresh" over-cap page that happens to sit in slot 0 but
    // introduces panel ids the previous page didn't have — refused.
    const impersonator: ConsolePage = {
      id: "overview",
      name: "Overview",
      panels: [
        ...previous.panels.slice(0, MAX_CONSOLE_PANELS_PER_PAGE),
        { id: "fresh-1", type: "markdown", content: {} },
        { id: "fresh-2", type: "markdown", content: {} },
      ],
      layout: [],
    };
    const err = validateConsolePagesDelta([impersonator], [previous]);
    expect(err).toContain("Too many panels");
  });

  it("mirrors backend delta rules: refuses over-cap page duplication via positional slot", () => {
    // Regression: a user with a grandfathered over-cap page "big"
    // used to be able to keep "big" AND insert a positional twin
    // that copied the same panel ids. Both new pages would appear
    // grandfathered individually (exact-id inheritance for "big",
    // positional inheritance for the twin) even though they share
    // the *same* previous slot. Delta validation now claims previous
    // slots by exact-id and refuses positional inheritance of an
    // already-claimed slot.
    const over = MAX_CONSOLE_PANELS_PER_PAGE + 4;
    const bigPanels = Array.from({ length: over }, (_v, i) => ({
      id: `p-${i}`,
      type: "markdown" as const,
      content: {},
    }));
    const previous: ConsolePage[] = [{ id: "big", name: "Big", panels: bigPanels, layout: [] }];

    // Positional twin ahead of the original — the exploit shape.
    const withDupBefore: ConsolePage[] = [
      { id: "big-clone", name: "Big clone", panels: bigPanels, layout: [] },
      { id: "big", name: "Big", panels: bigPanels, layout: [] },
    ];
    expect(validateConsolePagesDelta(withDupBefore, previous)).toContain("Too many panels");

    // Twin after the original — index 1 has no previous slot to
    // inherit either way, so this is also refused.
    const withDupAfter: ConsolePage[] = [
      { id: "big", name: "Big", panels: bigPanels, layout: [] },
      { id: "big-clone", name: "Big clone", panels: bigPanels, layout: [] },
    ];
    expect(validateConsolePagesDelta(withDupAfter, previous)).toContain("Too many panels");

    // Sanity: a legitimate rename without a duplicate still passes.
    const renamedOnly: ConsolePage[] = [{ id: "overview", name: "Overview", panels: bigPanels, layout: [] }];
    expect(validateConsolePagesDelta(renamedOnly, previous)).toBeNull();
  });

  it("mirrors backend delta rules: allows same page count above cap, refuses growth", () => {
    // 6 pages, previously 6 → allowed. Adding a 7th → refused.
    const buildPages = (n: number): ConsolePage[] =>
      Array.from({ length: n }, (_v, i) => ({
        id: `page-${i}`,
        name: `Page ${i}`,
        panels: [],
        layout: [],
      }));
    const previous = buildPages(MAX_CONSOLE_PAGES + 1);
    expect(validateConsolePagesDelta(buildPages(MAX_CONSOLE_PAGES + 1), previous)).toBeNull();
    expect(validateConsolePagesDelta(buildPages(MAX_CONSOLE_PAGES), previous)).toBeNull();
    const err = validateConsolePagesDelta(buildPages(MAX_CONSOLE_PAGES + 2), previous);
    expect(err).toContain("Too many pages");
  });

  it("wraps a legacy console into a single implicit `main` page", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: intro
      type: markdown
      content: {}
  layout:
    - i: intro
      x: 0
      y: 0
      w: 12
      h: 6
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.pages).toHaveLength(1);
    expect(result.data.spec.pages[0]!.id).toBe(DEFAULT_CONSOLE_PAGE_ID);
    expect(result.data.spec.pages[0]!.panels).toEqual([{ id: "intro", type: "markdown", content: {} }]);
  });
});
