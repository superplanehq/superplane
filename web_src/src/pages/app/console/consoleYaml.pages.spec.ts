import { describe, expect, it } from "vitest";

import {
  consoleToYaml,
  DEFAULT_CONSOLE_PAGE_ID,
  DEFAULT_CONSOLE_PAGE_NAME,
  MAX_CONSOLE_PAGES,
  MAX_CONSOLE_PANELS_PER_PAGE,
  parseConsoleYaml,
  parseConsoleYamlLenient,
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
