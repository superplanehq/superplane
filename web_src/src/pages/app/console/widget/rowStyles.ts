/**
 * Row-background palette + resolver for the Console Table widget.
 *
 * Authors configure a list of `WidgetRowStyle` rules. Each rule is one
 * field/op/value condition (same semantics as a `WidgetTableFilter`) mapped
 * to a tone from the curated palette. At render time the first matching
 * rule wins, mirroring how CSS cascades — earlier rules take precedence.
 *
 * The tone enum is decoupled from the raw Tailwind class so YAML stays
 * stable across utility-class refactors. The class map uses literal class
 * strings so Tailwind v4's JIT scanner picks them up.
 */

import { buildEnv } from "./celExpr";
import { evalCondition } from "./evalTableWhere";
import {
  WIDGET_FILTER_OPS,
  WIDGET_ROW_STYLE_TONES,
  type WidgetFilterOp,
  type WidgetRowStyle,
  type WidgetRowStyleTone,
} from "./types";

export const ROW_STYLE_CLASS: Record<WidgetRowStyleTone, string> = {
  dimmed: "bg-slate-100",
  yellow: "bg-yellow-100",
  "yellow-soft": "bg-yellow-50",
  orange: "bg-orange-100",
  "orange-soft": "bg-orange-50",
  red: "bg-red-100",
  "red-soft": "bg-red-50",
  blue: "bg-sky-100",
  "blue-soft": "bg-sky-50",
  green: "bg-emerald-100",
  "green-soft": "bg-emerald-50",
};

export const ROW_STYLE_LABEL: Record<WidgetRowStyleTone, string> = {
  dimmed: "Dimmed (slate)",
  "yellow-soft": "Yellow (soft)",
  yellow: "Yellow",
  "orange-soft": "Orange (soft)",
  orange: "Orange",
  "red-soft": "Red (soft)",
  red: "Red",
  "blue-soft": "Blue (soft)",
  blue: "Blue",
  "green-soft": "Green (soft)",
  green: "Green",
};

/**
 * Build a `(row) => className | undefined` resolver from the configured row
 * styles. Returns `undefined` when no rules are configured so callers can
 * short-circuit and skip the per-row work entirely. The CEL env is built
 * once and reused across rows (matches `applyTableWhere`).
 */
export function makeRowStyleResolver(
  rowStyles: WidgetRowStyle[] | undefined,
): ((row: Record<string, unknown>) => string | undefined) | undefined {
  if (!rowStyles || rowStyles.length === 0) return undefined;
  const env = buildEnv();
  return (row) => {
    for (const rule of rowStyles) {
      if (evalCondition(row, rule, env)) return ROW_STYLE_CLASS[rule.tone];
    }
    return undefined;
  };
}

/**
 * Coerce a persisted `render.rowStyles` value into the typed shape, dropping
 * entries with missing/invalid fields so a corrupted YAML import doesn't
 * crash the renderer. Returns `undefined` when no valid entries remain to
 * keep persisted YAML free of empty `rowStyles: []` stubs.
 */
export function normalizeWidgetRowStyles(raw: unknown): WidgetRowStyle[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  const out: WidgetRowStyle[] = [];
  for (const entry of raw) {
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) continue;
    const item = entry as Record<string, unknown>;
    const field = typeof item.field === "string" ? item.field : "";
    if (!field.trim()) continue;
    const op = typeof item.op === "string" ? item.op : "eq";
    if (!WIDGET_FILTER_OPS.includes(op as WidgetFilterOp)) continue;
    const tone = typeof item.tone === "string" ? item.tone : "";
    if (!WIDGET_ROW_STYLE_TONES.includes(tone as WidgetRowStyleTone)) continue;
    out.push({
      field,
      op: op as WidgetFilterOp,
      value: typeof item.value === "string" ? item.value : undefined,
      tone: tone as WidgetRowStyleTone,
    });
  }
  return out.length > 0 ? out : undefined;
}

/**
 * Strict per-rule validator surfaced by `validateTableContent`. Mirrors the
 * backend whitelist (`pkg/models/canvas_dashboard_yml.go`). Returns `null`
 * when the input is valid, or a human-readable message naming the offending
 * index + field.
 */
export function validateWidgetRowStyles(rowStyles: unknown): string | null {
  if (rowStyles == null) return null;
  if (!Array.isArray(rowStyles)) return "render.rowStyles must be an array.";
  for (let i = 0; i < rowStyles.length; i += 1) {
    const error = validateWidgetRowStyleRule(rowStyles[i], i);
    if (error) return error;
  }
  return null;
}

function validateWidgetRowStyleRule(raw: unknown, index: number): string | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return `render.rowStyles[${index}] must be an object.`;
  }
  const item = raw as Record<string, unknown>;
  if (typeof item.field !== "string" || item.field.trim() === "") {
    return `render.rowStyles[${index}].field must be a non-empty string.`;
  }
  if (typeof item.op !== "string" || !WIDGET_FILTER_OPS.includes(item.op as WidgetFilterOp)) {
    return `render.rowStyles[${index}].op is not supported.`;
  }
  if (typeof item.tone !== "string" || !WIDGET_ROW_STYLE_TONES.includes(item.tone as WidgetRowStyleTone)) {
    return `render.rowStyles[${index}].tone must be one of ${WIDGET_ROW_STYLE_TONES.join(", ")}.`;
  }
  if (item.value !== undefined && item.value !== null && typeof item.value !== "string") {
    return `render.rowStyles[${index}].value must be a string.`;
  }
  return null;
}
