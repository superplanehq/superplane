/**
 * Console YAML serialization helpers.
 *
 * Mirrors the Go validators in `pkg/yaml/console.go` so a YAML file
 * round-trips faithfully through both surfaces.
 *
 * The canonical schema supports two shapes:
 *   - Legacy (single-page): `spec.panels[]` + `spec.layout[]`.
 *   - Multi-page: `spec.pages[]` with `{id, name?, panels[], layout[]}`.
 *
 * A document that mixes both is rejected. Legacy inputs are wrapped into
 * one implicit `main` page so all downstream code works with the same
 * canonical `pages` list. Export follows a legacy-until-multi rule:
 * a console with 0 or 1 pages is emitted as the legacy shape so existing
 * single-page apps see zero diff noise; two or more pages emit `pages[]`.
 *
 * Import is replace-all (matches the backend commit path). Export is
 * deterministic.
 */

import * as yaml from "js-yaml";
import type { ConsoleLayoutItem, ConsolePage, ConsolePanel } from "@/hooks/useCanvasData";
import { PANEL_TYPES, isPanelType, validatePanelContent, type PanelType } from "./panelTypes";

export const CONSOLE_API_VERSION = "v1";
export const CONSOLE_KIND = "Console";

/**
 * Top-level panel types supported by the current Console YAML schema.
 * The validation source-of-truth lives in {@link PANEL_TYPES}; re-exported
 * here for ergonomics so YAML callers don't need a second import.
 */
export const SUPPORTED_PANEL_TYPES = PANEL_TYPES;
export type SupportedPanelType = PanelType;

// Caps mirror pkg/yaml/console.go. Existing consoles that already exceed
// them are grandfathered at read time; validation only fires on new
// imports / commits.
export const MAX_CONSOLE_PAGES = 5;
export const MAX_CONSOLE_PANELS_PER_PAGE = 20;
export const MAX_CONSOLE_PAYLOAD_BYTES = 1024 * 1024;

/** Id of the implicit page used when wrapping legacy single-page YAML. */
export const DEFAULT_CONSOLE_PAGE_ID = "main";
/** Human label paired with {@link DEFAULT_CONSOLE_PAGE_ID}. */
export const DEFAULT_CONSOLE_PAGE_NAME = "Main";

export type ConsoleYamlMetadata = {
  canvasId?: string;
  name?: string;
};

/**
 * Canonical (multi-page) parsed spec. Even legacy single-page YAML is
 * normalized into this shape so callers never branch on the source
 * layout. An empty console yields an empty `pages` array.
 */
export type ConsoleYamlSpec = {
  pages: ConsolePage[];
};

export type ConsoleYaml = {
  apiVersion: string;
  kind: string;
  metadata: ConsoleYamlMetadata;
  spec: ConsoleYamlSpec;
};

export type ConsoleYamlParseResult = { ok: true; data: ConsoleYaml } | { ok: false; error: string };
type ParseResult<T> = { ok: true; data: T } | { ok: false; error: string };

/**
 * Build canonical YAML text for a console. Empty and single-page consoles
 * export as the legacy `panels`/`layout` shape (backwards-compatible with
 * every existing app). Two or more pages export as `spec.pages[]`.
 */
export function consoleToYaml(input: { pages: ConsolePage[]; canvasId?: string; canvasName?: string }): string {
  const document = {
    apiVersion: CONSOLE_API_VERSION,
    kind: CONSOLE_KIND,
    metadata: {
      ...(input.canvasId ? { canvasId: input.canvasId } : {}),
      ...(input.canvasName ? { name: input.canvasName } : {}),
    },
    spec: consoleSpecForExport(input.pages),
  };

  return yaml.dump(document, {
    noRefs: true,
    lineWidth: 120,
    sortKeys: false,
  });
}

function consoleSpecForExport(pages: ConsolePage[]): Record<string, unknown> {
  // Legacy-until-multi: emit the pre-pages `panels`/`layout` shape only
  // when doing so is round-trippable — either the console is empty or
  // it has a single page that still uses the default id/name. If the
  // user renamed the sole page or picked a custom id, escape to the
  // multi-page shape so the rename survives the next save.
  if (pages.length === 0) {
    return { panels: [], layout: [] };
  }

  const first = pages[0]!;
  if (pages.length === 1 && isDefaultConsolePage(first)) {
    return {
      panels: first.panels.map(normalizePanelForExport),
      layout: first.layout.map(normalizeLayoutForExport),
    };
  }

  return {
    pages: pages.map((page) => {
      const entry: Record<string, unknown> = {
        id: page.id,
      };
      if (page.name) entry.name = page.name;
      entry.panels = page.panels.map(normalizePanelForExport);
      entry.layout = page.layout.map(normalizeLayoutForExport);
      return entry;
    }),
  };
}

function isDefaultConsolePage(page: ConsolePage): boolean {
  if (page.id !== DEFAULT_CONSOLE_PAGE_ID) return false;
  if (!page.name) return true;
  return page.name === DEFAULT_CONSOLE_PAGE_NAME;
}

/**
 * Parse a YAML string into a validated console import payload. Returns a
 * tagged union so callers can render the error message inline without
 * try/catch noise. Fails when structural rules (unknown fields, wrong
 * apiVersion, per-panel schema mismatches) are broken; also fails when a
 * cap is exceeded.
 *
 * Use this on **save / import paths** where the resulting document must
 * be safe to commit. For **read paths** that only render whatever the
 * backend already stores (which may be pre-cap grandfathered data), use
 * {@link parseConsoleYamlLenient} instead.
 */
export function parseConsoleYaml(text: string): ConsoleYamlParseResult {
  return parseConsoleYamlInternal(text, { validate: true });
}

/**
 * Same as {@link parseConsoleYaml} but skips the cap / uniqueness /
 * schema validation, so callers only pay for the structural YAML parse.
 * Intended for the render path: existing consoles that exceed the
 * per-page panel cap (grandfathered) still need to display; any
 * over-cap edit is rejected on the save path via {@link parseConsoleYaml}
 * or {@link validateConsolePages}.
 */
export function parseConsoleYamlLenient(text: string): ConsoleYamlParseResult {
  return parseConsoleYamlInternal(text, { validate: false });
}

function parseConsoleYamlInternal(text: string, options: { validate: boolean }): ConsoleYamlParseResult {
  const rootResult = parseConsoleRoot(text);
  if (!rootResult.ok) return rootResult;

  const pagesResult = parseConsolePages(rootResult.data.spec);
  if (!pagesResult.ok) return pagesResult;

  const document: ConsoleYaml = {
    apiVersion: CONSOLE_API_VERSION,
    kind: CONSOLE_KIND,
    metadata: rootResult.data.metadata,
    spec: { pages: pagesResult.data },
  };

  if (options.validate) {
    const validationError = validateConsolePages(document.spec.pages);
    if (validationError) {
      return { ok: false, error: validationError };
    }
  }

  return { ok: true, data: document };
}

function parseConsoleRoot(text: string): ParseResult<{
  metadata: ConsoleYamlMetadata;
  spec: Record<string, unknown>;
}> {
  const loaded = loadYamlRoot(text);
  if (!loaded.ok) return loaded;

  const rootError = validateRootHeader(loaded.data);
  if (rootError) return { ok: false, error: rootError };

  const metadata = parseMetadata(loaded.data.metadata);
  if (!metadata.ok) return metadata;

  const spec = parseSpec(loaded.data.spec);
  if (!spec.ok) return spec;

  return { ok: true, data: { metadata: metadata.data, spec: spec.data } };
}

function loadYamlRoot(text: string): ParseResult<Record<string, unknown>> {
  const trimmed = text.trim();
  if (!trimmed) return { ok: false, error: "Please provide a Console YAML definition." };

  let parsed: unknown;
  try {
    parsed = yaml.load(trimmed);
  } catch (e) {
    return { ok: false, error: `Invalid YAML syntax: ${e instanceof Error ? e.message : "Unknown error"}` };
  }

  if (!isPlainObject(parsed)) return { ok: false, error: "Console YAML must be an object at the root." };
  return { ok: true, data: parsed };
}

function validateRootHeader(root: Record<string, unknown>): string | null {
  const unknownKeys = unknownObjectKeys(root, ["apiVersion", "kind", "metadata", "spec"]);
  if (unknownKeys.length > 0) return `Unknown top-level field(s): ${unknownKeys.join(", ")}`;
  if (root.apiVersion !== CONSOLE_API_VERSION) {
    return `Unsupported apiVersion ${JSON.stringify(root.apiVersion)} (expected ${JSON.stringify(CONSOLE_API_VERSION)})`;
  }
  if (root.kind !== CONSOLE_KIND) {
    return `Unsupported kind ${JSON.stringify(root.kind)} (expected ${JSON.stringify(CONSOLE_KIND)})`;
  }
  return null;
}

function parseMetadata(raw: unknown): ParseResult<ConsoleYamlMetadata> {
  const metadata = raw ?? {};
  if (!isPlainObject(metadata)) return { ok: false, error: "metadata must be an object." };

  const unknownKeys = unknownObjectKeys(metadata, ["canvasId", "name"]);
  if (unknownKeys.length > 0) return { ok: false, error: `Unknown metadata field(s): ${unknownKeys.join(", ")}` };

  return {
    ok: true,
    data: {
      ...(typeof metadata.canvasId === "string" ? { canvasId: metadata.canvasId } : {}),
      ...(typeof metadata.name === "string" ? { name: metadata.name } : {}),
    },
  };
}

function parseSpec(raw: unknown): ParseResult<Record<string, unknown>> {
  if (!isPlainObject(raw)) return { ok: false, error: "spec must be an object." };

  const unknownKeys = unknownObjectKeys(raw, ["panels", "layout", "pages"]);
  if (unknownKeys.length > 0) return { ok: false, error: `Unknown spec field(s): ${unknownKeys.join(", ")}` };

  return { ok: true, data: raw };
}

// parseConsolePages normalizes the two accepted shapes into a canonical
// ConsolePage[]. Rejects documents that combine them so ambiguity can't
// slip through the round-trip pipeline.
function parseConsolePages(spec: Record<string, unknown>): ParseResult<ConsolePage[]> {
  const hasLegacy = spec.panels !== undefined || spec.layout !== undefined;
  const hasPages = spec.pages !== undefined;
  if (hasLegacy && hasPages) {
    return {
      ok: false,
      error: "spec.pages cannot be combined with top-level spec.panels or spec.layout.",
    };
  }

  if (hasPages) {
    return parsePagesArray(spec.pages);
  }

  const panelsResult = parsePanels(spec.panels);
  if (!panelsResult.ok) return panelsResult;
  const layoutResult = parseLayout(spec.layout);
  if (!layoutResult.ok) return layoutResult;

  if (panelsResult.data.length === 0 && layoutResult.data.length === 0) {
    return { ok: true, data: [] };
  }

  return {
    ok: true,
    data: [
      {
        id: DEFAULT_CONSOLE_PAGE_ID,
        name: DEFAULT_CONSOLE_PAGE_NAME,
        panels: panelsResult.data,
        layout: layoutResult.data,
      },
    ],
  };
}

function parsePagesArray(raw: unknown): ParseResult<ConsolePage[]> {
  if (raw === null || raw === undefined) return { ok: true, data: [] };
  if (!Array.isArray(raw)) return { ok: false, error: "spec.pages must be an array." };

  const pages: ConsolePage[] = [];
  for (let i = 0; i < raw.length; i += 1) {
    const item = raw[i];
    if (!isPlainObject(item)) return { ok: false, error: `spec.pages[${i}] must be an object.` };

    const unknownKeys = unknownObjectKeys(item, ["id", "name", "panels", "layout"]);
    if (unknownKeys.length > 0) {
      return { ok: false, error: `Unknown field(s) on spec.pages[${i}]: ${unknownKeys.join(", ")}` };
    }

    if (typeof item.id !== "string" || item.id.trim().length === 0) {
      return { ok: false, error: `spec.pages[${i}].id must be a non-empty string.` };
    }

    let name = "";
    if (item.name !== undefined && item.name !== null) {
      if (typeof item.name !== "string") return { ok: false, error: `spec.pages[${i}].name must be a string.` };
      name = item.name;
    }

    const panelsResult = parsePanels(item.panels, `spec.pages[${i}].panels`);
    if (!panelsResult.ok) return panelsResult;
    const layoutResult = parseLayout(item.layout, `spec.pages[${i}].layout`);
    if (!layoutResult.ok) return layoutResult;

    pages.push({ id: item.id, name, panels: panelsResult.data, layout: layoutResult.data });
  }
  return { ok: true, data: pages };
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function unknownObjectKeys(obj: Record<string, unknown>, allowed: string[]): string[] {
  return Object.keys(obj).filter((key) => !allowed.includes(key));
}

/**
 * Run structural validation on a canonical (pages) console. Matches the
 * backend `validateConsolePages` — page count cap, unique ids, per-page
 * panel cap, panel/layout consistency. Returns `null` when valid.
 */
export function validateConsolePages(pages: ConsolePage[]): string | null {
  if (pages.length > MAX_CONSOLE_PAGES) {
    return `Too many pages (max ${MAX_CONSOLE_PAGES}).`;
  }
  return validatePagesWith(pages, validateConsoleContent);
}

/**
 * Cap-independent structural validation. Runs everything
 * `validateConsolePages` does except the per-page panel-count cap and
 * the total-page-count cap. Used by import paths that tolerate
 * grandfathered over-cap consoles but still need to catch malformed
 * documents (duplicate ids, unsupported panel types, missing required
 * fields, broken layout references, oversized payload).
 */
export function validateConsolePagesStructural(pages: ConsolePage[]): string | null {
  return validatePagesWith(pages, validateConsoleContentStructural);
}

/**
 * Delta cap enforcement, mirroring the backend `ValidateConsolePagesDelta`
 * in `pkg/yaml/console.go`. Structural validation still runs on every
 * page (unique/non-empty ids, panel structure, layout references,
 * payload size). Cap rules:
 *
 *   - Page count: reject if new count > MAX_CONSOLE_PAGES AND new count
 *     > previous count. Grandfathered consoles with more than 5 pages
 *     may keep or shrink, but not grow further.
 *   - Per-page panel count: reject if the new count > MAX_CONSOLE_PANELS_PER_PAGE
 *     AND the new count > the previous count for either the same id OR
 *     the same positional slot (with panel-id-subset). The positional
 *     fallback keeps a grandfathered page valid across renames (e.g.
 *     `main` → `overview`) that do not add panels. Newly-introduced
 *     pages that share neither id nor position with any previous page
 *     must be at-or-under the cap on their first commit.
 *
 * Callers that don't have a baseline (fresh authoring) should pass
 * `[]` — that flips the rule to strict caps, matching the desired
 * "no over-cap on a canvas without grandfathered content" behavior.
 */
export function validateConsolePagesDelta(pages: ConsolePage[], previous: ConsolePage[]): string | null {
  const structuralError = validateConsolePagesStructural(pages);
  if (structuralError) return structuralError;

  if (pages.length > MAX_CONSOLE_PAGES && pages.length > previous.length) {
    return `Too many pages (max ${MAX_CONSOLE_PAGES}).`;
  }

  const previousPanelCountByID = new Map<string, number>();
  for (const page of previous) {
    previousPanelCountByID.set(page.id, page.panels.length);
  }

  for (let i = 0; i < pages.length; i += 1) {
    const page = pages[i];
    if (page.panels.length <= MAX_CONSOLE_PANELS_PER_PAGE) continue;
    if (isGrandfatheredOverCapPage(page, i, previous, previousPanelCountByID)) continue;
    return `Page ${JSON.stringify(page.id)}: Too many panels (max ${MAX_CONSOLE_PANELS_PER_PAGE} per page).`;
  }

  return null;
}

function isGrandfatheredOverCapPage(
  page: ConsolePage,
  index: number,
  previous: ConsolePage[],
  previousPanelCountByID: Map<string, number>,
): boolean {
  const byID = previousPanelCountByID.get(page.id);
  if (byID !== undefined && page.panels.length <= byID) return true;

  if (index >= previous.length) return false;
  const prevPage = previous[index];
  if (page.panels.length > prevPage.panels.length) return false;

  const prevPanelIDs = new Set(prevPage.panels.map((p) => p.id));
  for (const p of page.panels) {
    if (!prevPanelIDs.has(p.id)) return false;
  }
  return true;
}

function validatePagesWith(
  pages: ConsolePage[],
  perPage: (panels: ConsolePanel[], layout: ConsoleLayoutItem[]) => string | null,
): string | null {
  const pageIds = new Set<string>();
  for (let i = 0; i < pages.length; i += 1) {
    const page = pages[i];
    if (!page.id.trim()) return `pages[${i}].id is required.`;
    if (pageIds.has(page.id)) return `Duplicate page id ${JSON.stringify(page.id)}.`;
    pageIds.add(page.id);

    const contentError = perPage(page.panels, page.layout);
    if (contentError) return `Page ${JSON.stringify(page.id)}: ${contentError}`;
  }
  return null;
}

/**
 * Structural validation for the panels + layout of a single page.
 * Reused for both the legacy shape (wrapped into an implicit page) and
 * each individual page in the multi-page shape.
 */
export function validateConsoleContent(panels: ConsolePanel[], layout: ConsoleLayoutItem[]): string | null {
  if (panels.length > MAX_CONSOLE_PANELS_PER_PAGE) {
    return `Too many panels (max ${MAX_CONSOLE_PANELS_PER_PAGE} per page).`;
  }
  return validateConsoleContentStructural(panels, layout);
}

function validateConsoleContentStructural(panels: ConsolePanel[], layout: ConsoleLayoutItem[]): string | null {
  const panelIdsResult = validatePanels(panels);
  if (!panelIdsResult.ok) return panelIdsResult.error;

  const payloadError = validatePanelsPayloadSize(panels);
  if (payloadError) return payloadError;

  return validateLayoutReferences(layout, panelIdsResult.data);
}

function validatePanels(panels: ConsolePanel[]): ParseResult<Set<string>> {
  const panelIds = new Set<string>();
  for (const panel of panels) {
    if (!panel.id) return { ok: false, error: "Panel id is required." };
    if (!panel.type) return { ok: false, error: `Panel ${JSON.stringify(panel.id)} type is required.` };
    if (!isPanelType(panel.type)) {
      return {
        ok: false,
        error: `Panel ${JSON.stringify(panel.id)} has unsupported type ${JSON.stringify(panel.type)}.`,
      };
    }
    if (panelIds.has(panel.id)) {
      return { ok: false, error: `Duplicate panel id ${JSON.stringify(panel.id)}.` };
    }
    panelIds.add(panel.id);

    const contentError = validatePanelContent(panel.type, panel.content);
    if (contentError) {
      return { ok: false, error: `Panel ${JSON.stringify(panel.id)} ${contentError}` };
    }
  }

  return { ok: true, data: panelIds };
}

function validatePanelsPayloadSize(panels: ConsolePanel[]): string | null {
  const encodedSize = byteLengthUtf8(JSON.stringify(panels));
  if (encodedSize > MAX_CONSOLE_PAYLOAD_BYTES) {
    return `Panels payload exceeds ${MAX_CONSOLE_PAYLOAD_BYTES} bytes.`;
  }
  return null;
}

function validateLayoutReferences(layout: ConsoleLayoutItem[], panelIds: Set<string>): string | null {
  const layoutIds = new Set<string>();
  for (const item of layout) {
    if (!item.i) return "Layout item i is required.";
    if (layoutIds.has(item.i)) return `Duplicate layout id ${JSON.stringify(item.i)}.`;
    layoutIds.add(item.i);

    if (!panelIds.has(item.i)) {
      return `Layout item ${JSON.stringify(item.i)} does not reference any panel.`;
    }
    if (item.w <= 0 || item.h <= 0) {
      return `Layout item ${JSON.stringify(item.i)} must have positive width and height.`;
    }
    if (item.x < 0 || item.y < 0) {
      return `Layout item ${JSON.stringify(item.i)} must have non-negative x and y.`;
    }
  }

  return null;
}

function parsePanels(
  raw: unknown,
  scope: string = "spec.panels",
): { ok: true; data: ConsolePanel[] } | { ok: false; error: string } {
  if (raw === undefined || raw === null) return { ok: true, data: [] };
  if (!Array.isArray(raw)) return { ok: false, error: `${scope} must be an array.` };

  const panels: ConsolePanel[] = [];
  for (let i = 0; i < raw.length; i += 1) {
    const parsed = parsePanelEntry(raw[i], i, scope);
    if (!parsed.ok) return parsed;
    panels.push(parsed.data);
  }
  return { ok: true, data: panels };
}

function parsePanelEntry(value: unknown, index: number, scope: string): ParseResult<ConsolePanel> {
  if (!isPlainObject(value)) return { ok: false, error: `${scope}[${index}] must be an object.` };

  const unknownKeys = unknownObjectKeys(value, ["id", "type", "content"]);
  if (unknownKeys.length > 0) {
    return { ok: false, error: `Unknown field(s) on panel ${index}: ${unknownKeys.join(", ")}` };
  }
  if (typeof value.id !== "string") return { ok: false, error: `${scope}[${index}].id must be a string.` };
  if (typeof value.type !== "string") return { ok: false, error: `${scope}[${index}].type must be a string.` };

  const contentResult = parsePanelContent(value.content, index, scope);
  if (!contentResult.ok) return contentResult;

  return { ok: true, data: { id: value.id, type: value.type, content: contentResult.data } };
}

function parsePanelContent(raw: unknown, index: number, scope: string): ParseResult<Record<string, unknown>> {
  if (raw === undefined || raw === null) return { ok: true, data: {} };
  if (!isPlainObject(raw)) return { ok: false, error: `${scope}[${index}].content must be an object.` };
  return { ok: true, data: raw };
}

function parseLayout(
  raw: unknown,
  scope: string = "spec.layout",
): { ok: true; data: ConsoleLayoutItem[] } | { ok: false; error: string } {
  if (raw === undefined || raw === null) return { ok: true, data: [] };
  if (!Array.isArray(raw)) return { ok: false, error: `${scope} must be an array.` };

  const layout: ConsoleLayoutItem[] = [];
  for (let i = 0; i < raw.length; i += 1) {
    const item = parseLayoutItem(raw[i], i, scope);
    if (!item.ok) return item;
    layout.push(item.data);
  }
  return { ok: true, data: layout };
}

function parseLayoutItem(raw: unknown, index: number, scope: string): ParseResult<ConsoleLayoutItem> {
  if (!isPlainObject(raw)) return { ok: false, error: `${scope}[${index}] must be an object.` };

  const unknownKeys = unknownObjectKeys(raw, ["i", "x", "y", "w", "h", "minW", "minH"]);
  if (unknownKeys.length > 0) {
    return { ok: false, error: `Unknown field(s) on layout ${index}: ${unknownKeys.join(", ")}` };
  }
  if (typeof raw.i !== "string") return { ok: false, error: `${scope}[${index}].i must be a string.` };

  const numericError = validateLayoutItemNumbers(raw, index, scope);
  if (numericError) return { ok: false, error: numericError };

  const item: ConsoleLayoutItem = {
    i: raw.i,
    x: raw.x as number,
    y: raw.y as number,
    w: raw.w as number,
    h: raw.h as number,
  };
  if (typeof raw.minW === "number") item.minW = raw.minW;
  if (typeof raw.minH === "number") item.minH = raw.minH;
  return { ok: true, data: item };
}

function validateLayoutItemNumbers(entry: Record<string, unknown>, index: number, scope: string): string | null {
  const numericFields: Array<["x" | "y" | "w" | "h" | "minW" | "minH", boolean]> = [
    ["x", true],
    ["y", true],
    ["w", true],
    ["h", true],
    ["minW", false],
    ["minH", false],
  ];

  for (const [field, required] of numericFields) {
    const value = entry[field];
    if (value === undefined) {
      if (required) return `${scope}[${index}].${field} is required.`;
      continue;
    }
    if (typeof value !== "number" || !Number.isFinite(value)) return `${scope}[${index}].${field} must be a number.`;
  }
  return null;
}

function normalizePanelForExport(panel: ConsolePanel): ConsolePanel {
  return {
    id: panel.id,
    type: panel.type,
    content: panel.content ?? {},
  };
}

function normalizeLayoutForExport(item: ConsoleLayoutItem): ConsoleLayoutItem {
  const out: ConsoleLayoutItem = {
    i: item.i,
    x: item.x,
    y: item.y,
    w: item.w,
    h: item.h,
  };
  if (item.minW !== undefined) out.minW = item.minW;
  if (item.minH !== undefined) out.minH = item.minH;
  return out;
}

function byteLengthUtf8(s: string): number {
  if (typeof TextEncoder !== "undefined") return new TextEncoder().encode(s).length;
  let total = 0;
  for (let i = 0; i < s.length; i += 1) {
    const code = s.charCodeAt(i);
    if (code < 0x80) total += 1;
    else if (code < 0x800) total += 2;
    else if (code >= 0xd800 && code <= 0xdbff) {
      total += 4;
      i += 1;
    } else total += 3;
  }
  return total;
}

export function consoleYamlFilename(canvasName?: string): string {
  const safe = (canvasName || "console")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
  return `${safe || "console"}-console.yaml`;
}

/**
 * Flatten every page's panels into one array. Convenience for callers
 * that treat the console as one collection — draft/diff logic, staging
 * indicators, defaultTab.ts, etc. Preserves page order and per-page panel
 * order for stable comparisons.
 */
export function flattenConsolePanels(pages: ConsolePage[]): ConsolePanel[] {
  const out: ConsolePanel[] = [];
  for (const page of pages) {
    for (const panel of page.panels) out.push(panel);
  }
  return out;
}

/**
 * Flatten every page's layout items into one array. Same rationale as
 * {@link flattenConsolePanels}.
 */
export function flattenConsoleLayout(pages: ConsolePage[]): ConsoleLayoutItem[] {
  const out: ConsoleLayoutItem[] = [];
  for (const page of pages) {
    for (const item of page.layout) out.push(item);
  }
  return out;
}
