/**
 * Dashboard YAML serialization helpers.
 *
 * Mirrors the canonical schema produced/consumed by `pkg/models/canvas_dashboard_yml.go`
 * so that a YAML file round-trips faithfully through both surfaces.
 *
 * The canonical schema is:
 *   apiVersion: v1
 *   kind: Dashboard
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
export const DASHBOARD_KIND = "Dashboard";

/**
 * Top-level panel types supported by the current Dashboard YAML schema.
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

/**
 * Build canonical YAML text for a dashboard. The result matches the layout
 * produced by `DashboardToYML` on the backend: stable key order, no nullish
 * `minW`/`minH` keys, empty dashboards still produce a valid empty `spec`.
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
 * Parse a YAML string into a validated dashboard import payload. Returns a
 * tagged union so callers can render the error message inline without
 * try/catch noise.
 */
export function parseDashboardYaml(text: string): DashboardYamlParseResult {
  const trimmed = text.trim();
  if (!trimmed) {
    return { ok: false, error: "Please provide a Dashboard YAML definition." };
  }

  let parsed: unknown;
  try {
    parsed = yaml.load(trimmed);
  } catch (e) {
    return {
      ok: false,
      error: `Invalid YAML syntax: ${e instanceof Error ? e.message : "Unknown error"}`,
    };
  }

  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { ok: false, error: "Dashboard YAML must be an object at the root." };
  }

  const root = parsed as Record<string, unknown>;
  const unknownKeys = Object.keys(root).filter((key) => !["apiVersion", "kind", "metadata", "spec"].includes(key));
  if (unknownKeys.length > 0) {
    return { ok: false, error: `Unknown top-level field(s): ${unknownKeys.join(", ")}` };
  }

  if (root.apiVersion !== DASHBOARD_API_VERSION) {
    return {
      ok: false,
      error: `Unsupported apiVersion ${JSON.stringify(root.apiVersion)} (expected ${JSON.stringify(DASHBOARD_API_VERSION)})`,
    };
  }
  if (root.kind !== DASHBOARD_KIND) {
    return {
      ok: false,
      error: `Unsupported kind ${JSON.stringify(root.kind)} (expected ${JSON.stringify(DASHBOARD_KIND)})`,
    };
  }

  const metadata = root.metadata ?? {};
  if (typeof metadata !== "object" || metadata === null || Array.isArray(metadata)) {
    return { ok: false, error: "metadata must be an object." };
  }
  const metadataObj = metadata as Record<string, unknown>;
  const metadataUnknown = Object.keys(metadataObj).filter((key) => !["canvasId", "name"].includes(key));
  if (metadataUnknown.length > 0) {
    return { ok: false, error: `Unknown metadata field(s): ${metadataUnknown.join(", ")}` };
  }

  const spec = root.spec;
  if (!spec || typeof spec !== "object" || Array.isArray(spec)) {
    return { ok: false, error: "spec must be an object." };
  }
  const specObj = spec as Record<string, unknown>;
  const specUnknown = Object.keys(specObj).filter((key) => !["panels", "layout"].includes(key));
  if (specUnknown.length > 0) {
    return { ok: false, error: `Unknown spec field(s): ${specUnknown.join(", ")}` };
  }

  const panelsResult = parsePanels(specObj.panels);
  if (!panelsResult.ok) return panelsResult;

  const layoutResult = parseLayout(specObj.layout);
  if (!layoutResult.ok) return layoutResult;

  const document: DashboardYaml = {
    apiVersion: DASHBOARD_API_VERSION,
    kind: DASHBOARD_KIND,
    metadata: {
      ...(typeof metadataObj.canvasId === "string" ? { canvasId: metadataObj.canvasId } : {}),
      ...(typeof metadataObj.name === "string" ? { name: metadataObj.name } : {}),
    },
    spec: { panels: panelsResult.data, layout: layoutResult.data },
  };

  const validationError = validateDashboardContent(document.spec.panels, document.spec.layout);
  if (validationError) {
    return { ok: false, error: validationError };
  }

  return { ok: true, data: document };
}

/**
 * Run the shared structural validation matching the backend's
 * `ValidateDashboardContent`. Returns `null` when the input is valid.
 */
export function validateDashboardContent(panels: DashboardPanel[], layout: DashboardLayoutItem[]): string | null {
  if (panels.length > MAX_DASHBOARD_PANELS) {
    return `Too many panels (max ${MAX_DASHBOARD_PANELS}).`;
  }

  const panelIds = new Set<string>();
  for (const panel of panels) {
    if (!panel.id) return "Panel id is required.";
    if (!panel.type) return `Panel ${JSON.stringify(panel.id)} type is required.`;
    if (!isPanelType(panel.type)) {
      return `Panel ${JSON.stringify(panel.id)} has unsupported type ${JSON.stringify(panel.type)}.`;
    }
    if (panelIds.has(panel.id)) {
      return `Duplicate panel id ${JSON.stringify(panel.id)}.`;
    }
    panelIds.add(panel.id);

    const contentError = validatePanelContent(panel.type, panel.content);
    if (contentError) {
      return `Panel ${JSON.stringify(panel.id)} ${contentError}`;
    }
  }

  const encodedSize = byteLengthUtf8(JSON.stringify(panels));
  if (encodedSize > MAX_DASHBOARD_PAYLOAD_BYTES) {
    return `Panels payload exceeds ${MAX_DASHBOARD_PAYLOAD_BYTES} bytes.`;
  }

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
    const value = raw[i];
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      return { ok: false, error: `spec.layout[${i}] must be an object.` };
    }
    const entry = value as Record<string, unknown>;
    const allowed = ["i", "x", "y", "w", "h", "minW", "minH"];
    const unknownKeys = Object.keys(entry).filter((key) => !allowed.includes(key));
    if (unknownKeys.length > 0) {
      return { ok: false, error: `Unknown field(s) on layout ${i}: ${unknownKeys.join(", ")}` };
    }
    if (typeof entry.i !== "string") return { ok: false, error: `spec.layout[${i}].i must be a string.` };
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
        if (required) return { ok: false, error: `spec.layout[${i}].${field} is required.` };
        continue;
      }
      if (typeof value !== "number" || !Number.isFinite(value)) {
        return { ok: false, error: `spec.layout[${i}].${field} must be a number.` };
      }
    }
    const item: DashboardLayoutItem = {
      i: entry.i,
      x: entry.x as number,
      y: entry.y as number,
      w: entry.w as number,
      h: entry.h as number,
    };
    if (typeof entry.minW === "number") item.minW = entry.minW;
    if (typeof entry.minH === "number") item.minH = entry.minH;
    layout.push(item);
  }
  return { ok: true, data: layout };
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
  const safe = (canvasName || "dashboard")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
  return `${safe || "dashboard"}-dashboard.yaml`;
}
