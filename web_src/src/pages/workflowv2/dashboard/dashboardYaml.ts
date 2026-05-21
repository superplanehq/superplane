/**
 * Console YAML serialization helpers.
 *
 * The product surfaces this as "Console". Go validators in
 * `pkg/models/canvas_dashboard_yml.go` accept both `kind: Console` (canonical)
 * and legacy `kind: Dashboard` on import; export uses Console on both surfaces.
 *
 * The canonical schema is:
 *   apiVersion: v1
 *   kind: Console        # legacy "Dashboard" is still accepted on import
 *   metadata:
 *     canvasId?: <uuid>
 *     name?: <display-only>
 *   spec:
 *     panels: DashboardPanel[]
 *     layout: DashboardLayoutItem[]
 *
 * Import is replace-all (matches `UpdateCanvasDashboard`). Export is deterministic.
 */

import * as yaml from "js-yaml";
import type { DashboardLayoutItem, DashboardPanel } from "@/hooks/useCanvasData";
import { PANEL_TYPES, isPanelType, validatePanelContent, type PanelType } from "./panelTypes";

export const DASHBOARD_API_VERSION = "v1";
export const DASHBOARD_KIND = "Console";
// Legacy kind value accepted on import for back-compat with files exported
// before the Console rename.
const LEGACY_DASHBOARD_KIND = "Dashboard";

/**
 * Top-level panel types supported by the current Console YAML schema.
 * The validation source-of-truth lives in {@link PANEL_TYPES}; re-exported
 * here for ergonomics so YAML callers don't need a second import.
 */
export const SUPPORTED_PANEL_TYPES = PANEL_TYPES;
export type SupportedPanelType = PanelType;

export const MAX_DASHBOARD_PANELS = 50;
export const MAX_DASHBOARD_PAYLOAD_BYTES = 1024 * 1024;

export type DashboardYamlMetadata = {
  canvasId?: string;
  name?: string;
};

export type DashboardYamlSpec = {
  panels: DashboardPanel[];
  layout: DashboardLayoutItem[];
};

export type DashboardYaml = {
  apiVersion: string;
  kind: string;
  metadata: DashboardYamlMetadata;
  spec: DashboardYamlSpec;
};

export type DashboardYamlParseResult = { ok: true; data: DashboardYaml } | { ok: false; error: string };
type ParseResult<T> = { ok: true; data: T } | { ok: false; error: string };

/**
 * Build canonical YAML text for a console. The result matches the layout
 * produced by `DashboardToYML` on the backend: stable key order, no nullish
 * `minW`/`minH` keys, empty consoles still produce a valid empty `spec`.
 */
export function dashboardToYaml(input: {
  panels: DashboardPanel[];
  layout: DashboardLayoutItem[];
  canvasId?: string;
  canvasName?: string;
}): string {
  const document: DashboardYaml = {
    apiVersion: DASHBOARD_API_VERSION,
    kind: DASHBOARD_KIND,
    metadata: {
      ...(input.canvasId ? { canvasId: input.canvasId } : {}),
      ...(input.canvasName ? { name: input.canvasName } : {}),
    },
    spec: {
      panels: input.panels.map(normalizePanelForExport),
      layout: input.layout.map(normalizeLayoutForExport),
    },
  };

  return yaml.dump(document, {
    noRefs: true,
    lineWidth: 120,
    sortKeys: false,
  });
}

/**
 * Parse a YAML string into a validated console import payload. Returns a
 * tagged union so callers can render the error message inline without
 * try/catch noise.
 */
export function parseDashboardYaml(text: string): DashboardYamlParseResult {
  const rootResult = parseDashboardRoot(text);
  if (!rootResult.ok) return rootResult;

  const panelsResult = parsePanels(rootResult.data.spec.panels);
  if (!panelsResult.ok) return panelsResult;

  const layoutResult = parseLayout(rootResult.data.spec.layout);
  if (!layoutResult.ok) return layoutResult;

  const document: DashboardYaml = {
    apiVersion: DASHBOARD_API_VERSION,
    kind: DASHBOARD_KIND,
    metadata: rootResult.data.metadata,
    spec: { panels: panelsResult.data, layout: layoutResult.data },
  };

  const validationError = validateDashboardContent(document.spec.panels, document.spec.layout);
  if (validationError) {
    return { ok: false, error: validationError };
  }

  return { ok: true, data: document };
}

function parseDashboardRoot(text: string): ParseResult<{
  metadata: DashboardYamlMetadata;
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
  if (root.apiVersion !== DASHBOARD_API_VERSION) {
    return `Unsupported apiVersion ${JSON.stringify(root.apiVersion)} (expected ${JSON.stringify(DASHBOARD_API_VERSION)})`;
  }
  if (root.kind !== DASHBOARD_KIND && root.kind !== LEGACY_DASHBOARD_KIND) {
    return `Unsupported kind ${JSON.stringify(root.kind)} (expected ${JSON.stringify(DASHBOARD_KIND)})`;
  }
  return null;
}

function parseMetadata(raw: unknown): ParseResult<DashboardYamlMetadata> {
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

  const unknownKeys = unknownObjectKeys(raw, ["panels", "layout"]);
  if (unknownKeys.length > 0) return { ok: false, error: `Unknown spec field(s): ${unknownKeys.join(", ")}` };

  return { ok: true, data: raw };
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function unknownObjectKeys(obj: Record<string, unknown>, allowed: string[]): string[] {
  return Object.keys(obj).filter((key) => !allowed.includes(key));
}

/**
 * Run the shared structural validation matching the backend's
 * `ValidateDashboardContent`. Returns `null` when the input is valid.
 */
export function validateDashboardContent(panels: DashboardPanel[], layout: DashboardLayoutItem[]): string | null {
  if (panels.length > MAX_DASHBOARD_PANELS) {
    return `Too many panels (max ${MAX_DASHBOARD_PANELS}).`;
  }

  const panelIdsResult = validatePanels(panels);
  if (!panelIdsResult.ok) return panelIdsResult.error;

  const payloadError = validatePanelsPayloadSize(panels);
  if (payloadError) return payloadError;

  return validateLayoutReferences(layout, panelIdsResult.data);
}

function validatePanels(panels: DashboardPanel[]): ParseResult<Set<string>> {
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

function validatePanelsPayloadSize(panels: DashboardPanel[]): string | null {
  const encodedSize = byteLengthUtf8(JSON.stringify(panels));
  if (encodedSize > MAX_DASHBOARD_PAYLOAD_BYTES) {
    return `Panels payload exceeds ${MAX_DASHBOARD_PAYLOAD_BYTES} bytes.`;
  }
  return null;
}

function validateLayoutReferences(layout: DashboardLayoutItem[], panelIds: Set<string>): string | null {
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

function parsePanels(raw: unknown): { ok: true; data: DashboardPanel[] } | { ok: false; error: string } {
  if (raw === undefined || raw === null) return { ok: true, data: [] };
  if (!Array.isArray(raw)) return { ok: false, error: "spec.panels must be an array." };

  const panels: DashboardPanel[] = [];
  for (let i = 0; i < raw.length; i += 1) {
    const value = raw[i];
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      return { ok: false, error: `spec.panels[${i}] must be an object.` };
    }
    const entry = value as Record<string, unknown>;
    const unknownKeys = Object.keys(entry).filter((key) => !["id", "type", "content"].includes(key));
    if (unknownKeys.length > 0) {
      return { ok: false, error: `Unknown field(s) on panel ${i}: ${unknownKeys.join(", ")}` };
    }
    if (typeof entry.id !== "string") return { ok: false, error: `spec.panels[${i}].id must be a string.` };
    if (typeof entry.type !== "string") return { ok: false, error: `spec.panels[${i}].type must be a string.` };

    let content: Record<string, unknown> = {};
    if (entry.content !== undefined && entry.content !== null) {
      if (typeof entry.content !== "object" || Array.isArray(entry.content)) {
        return { ok: false, error: `spec.panels[${i}].content must be an object.` };
      }
      content = entry.content as Record<string, unknown>;
    }
    panels.push({ id: entry.id, type: entry.type, content });
  }
  return { ok: true, data: panels };
}

function parseLayout(raw: unknown): { ok: true; data: DashboardLayoutItem[] } | { ok: false; error: string } {
  if (raw === undefined || raw === null) return { ok: true, data: [] };
  if (!Array.isArray(raw)) return { ok: false, error: "spec.layout must be an array." };

  const layout: DashboardLayoutItem[] = [];
  for (let i = 0; i < raw.length; i += 1) {
    const item = parseLayoutItem(raw[i], i);
    if (!item.ok) return item;
    layout.push(item.data);
  }
  return { ok: true, data: layout };
}

function parseLayoutItem(raw: unknown, index: number): ParseResult<DashboardLayoutItem> {
  if (!isPlainObject(raw)) return { ok: false, error: `spec.layout[${index}] must be an object.` };

  const unknownKeys = unknownObjectKeys(raw, ["i", "x", "y", "w", "h", "minW", "minH"]);
  if (unknownKeys.length > 0) {
    return { ok: false, error: `Unknown field(s) on layout ${index}: ${unknownKeys.join(", ")}` };
  }
  if (typeof raw.i !== "string") return { ok: false, error: `spec.layout[${index}].i must be a string.` };

  const numericError = validateLayoutItemNumbers(raw, index);
  if (numericError) return { ok: false, error: numericError };

  const item: DashboardLayoutItem = {
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

function validateLayoutItemNumbers(entry: Record<string, unknown>, index: number): string | null {
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
      if (required) return `spec.layout[${index}].${field} is required.`;
      continue;
    }
    if (typeof value !== "number" || !Number.isFinite(value)) return `spec.layout[${index}].${field} must be a number.`;
  }
  return null;
}

function normalizePanelForExport(panel: DashboardPanel): DashboardPanel {
  return {
    id: panel.id,
    type: panel.type,
    content: panel.content ?? {},
  };
}

function normalizeLayoutForExport(item: DashboardLayoutItem): DashboardLayoutItem {
  const out: DashboardLayoutItem = {
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

export function dashboardYamlFilename(canvasName?: string): string {
  const safe = (canvasName || "console")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
  return `${safe || "console"}-console.yaml`;
}
