/**
 * Typed content shape, template, and validator for the kanban `board`
 * dashboard panel.
 *
 * Kept in its own module so `panelTypes.ts` stays under the shared lint
 * budget. Both the template seed and the validator are re-exported from
 * `panelTypes.ts` so callers keep going through a single entry point.
 *
 * The board panel reuses the same data-fetch pipeline as the table panel
 * (`useWidgetData` + `applyTableWhere`); the only difference is the
 * renderer, which groups filtered rows into configured lanes by a scalar
 * `groupBy` field. See `WidgetBoardRender` in `widget/types.ts` for the
 * render shape. Keep this in lockstep with `validateBoardPanelContent` in
 * `pkg/yaml/console.go`.
 */

import { asObject, optionalStringError } from "./panelContentValidation";
import type { TablePanelDataSource } from "./panelTypes";
import { validateDataSource, validateSort } from "./panelTypes";
import {
  WIDGET_BOARD_LANE_COLORS,
  WIDGET_FILTER_OPS,
  type WidgetBoardCard,
  type WidgetBoardLane,
  type WidgetBoardLaneColor,
  type WidgetBoardRender,
  type WidgetRowAction,
  type WidgetTableColumn,
  type WidgetTableFilter,
} from "./widget/types";
import { normalizeRowAction } from "./widget/types";

export interface BoardPanelContent {
  title?: string;
  dataSource: TablePanelDataSource;
  render: WidgetBoardRender;
}

/** Default content for a newly added `board` panel. */
export function templateForBoardPanel(defaultTitle?: string): BoardPanelContent {
  return {
    title: defaultTitle ?? "",
    dataSource: { kind: "memory", namespace: "" },
    render: {
      kind: "board",
      groupBy: "status",
      lanes: [{ value: "Todo" }, { value: "In Progress" }, { value: "Done", color: "green" }],
      card: { titleField: "title" },
    },
  };
}

/**
 * Coerce a persisted board panel body into the typed shape. Behaves like
 * `normalizeTablePanelContent`: unknown fields are dropped, invalid list
 * entries are filtered out, and the returned value is safe to feed into
 * the renderer even when the source YAML is partially broken.
 */
export function normalizeBoardPanelContent(raw: Record<string, unknown> | undefined): BoardPanelContent {
  const r = raw ?? {};
  const renderRaw = asObject(r.render) ?? {};
  const cardRaw = asObject(renderRaw.card) ?? {};

  return {
    title: typeof r.title === "string" ? r.title : "",
    dataSource: normalizeBoardDataSource(r.dataSource),
    render: {
      kind: "board",
      groupBy: typeof renderRaw.groupBy === "string" ? renderRaw.groupBy : "",
      lanes: normalizeLanes(renderRaw.lanes),
      otherLane: renderRaw.otherLane === true ? true : undefined,
      card: normalizeCard(cardRaw),
      where: normalizeBoardWhere(renderRaw.where),
      sort: normalizeBoardSort(renderRaw.sort),
      rowActions: normalizeBoardRowActions(renderRaw.rowActions),
      emptyMessage: typeof renderRaw.emptyMessage === "string" ? renderRaw.emptyMessage : undefined,
    },
  };
}

function normalizeBoardDataSource(raw: unknown): TablePanelDataSource {
  const obj = asObject(raw);
  if (!obj) return { kind: "memory", namespace: "" };
  switch (obj.kind) {
    case "memory":
      return {
        kind: "memory",
        namespace: typeof obj.namespace === "string" ? obj.namespace : "",
        fieldPath: typeof obj.fieldPath === "string" ? obj.fieldPath : undefined,
      };
    case "executions":
      return {
        kind: "executions",
        node: typeof obj.node === "string" ? obj.node : undefined,
        limit: typeof obj.limit === "number" ? obj.limit : undefined,
      };
    case "runs":
      return { kind: "runs", limit: typeof obj.limit === "number" ? obj.limit : undefined };
    default:
      return { kind: "memory", namespace: "" };
  }
}

function normalizeLanes(raw: unknown): WidgetBoardLane[] {
  if (!Array.isArray(raw)) return [];
  const out: WidgetBoardLane[] = [];
  for (const entry of raw) {
    const obj = asObject(entry);
    if (!obj) continue;
    const value = typeof obj.value === "string" ? obj.value : "";
    if (!value.trim()) continue;
    const color =
      typeof obj.color === "string" && WIDGET_BOARD_LANE_COLORS.includes(obj.color as WidgetBoardLaneColor)
        ? (obj.color as WidgetBoardLaneColor)
        : undefined;
    out.push({
      value,
      label: typeof obj.label === "string" ? obj.label : undefined,
      color,
    });
  }
  return out;
}

function normalizeCard(raw: Record<string, unknown>): WidgetBoardCard {
  return {
    titleField: typeof raw.titleField === "string" ? raw.titleField : "",
    fields: normalizeCardFields(raw.fields),
  };
}

function normalizeCardFields(raw: unknown): WidgetTableColumn[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  const out: WidgetTableColumn[] = [];
  for (const entry of raw) {
    const obj = asObject(entry);
    if (!obj) continue;
    const field = typeof obj.field === "string" ? obj.field : "";
    if (!field.trim()) continue;
    out.push({
      field,
      label: typeof obj.label === "string" ? obj.label : undefined,
      format: typeof obj.format === "string" ? (obj.format as WidgetTableColumn["format"]) : undefined,
      show: typeof obj.show === "string" ? obj.show : undefined,
      href: typeof obj.href === "string" ? obj.href : undefined,
    });
  }
  return out.length > 0 ? out : undefined;
}

function normalizeBoardWhere(raw: unknown): WidgetTableFilter[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  const out: WidgetTableFilter[] = [];
  for (const entry of raw) {
    const obj = asObject(entry);
    if (!obj) continue;
    const field = typeof obj.field === "string" ? obj.field : "";
    const op = typeof obj.op === "string" ? obj.op : "";
    if (!field.trim() || !WIDGET_FILTER_OPS.includes(op as WidgetTableFilter["op"])) continue;
    out.push({
      field,
      op: op as WidgetTableFilter["op"],
      value: typeof obj.value === "string" ? obj.value : undefined,
    });
  }
  return out.length > 0 ? out : undefined;
}

function normalizeBoardSort(raw: unknown): { field: string; order?: "asc" | "desc" } | undefined {
  const obj = asObject(raw);
  if (!obj) return undefined;
  const field = typeof obj.field === "string" ? obj.field.trim() : "";
  if (!field) return undefined;
  const order: "asc" | "desc" | undefined =
    obj.order === "desc" ? "desc" : obj.order === "asc" ? "asc" : undefined;
  return order ? { field, order } : { field };
}

function normalizeBoardRowActions(raw: unknown): WidgetRowAction[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  const out: WidgetRowAction[] = [];
  for (const entry of raw) {
    const action = normalizeRowAction(entry);
    if (action) out.push(action);
  }
  return out.length > 0 ? out : undefined;
}

/** Validate the persisted `board` content. Returns null when valid. */
export function validateBoardContent(content: unknown): string | null {
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  const titleError = optionalStringError("content.title", obj.title);
  if (titleError) return titleError;

  const dsError = validateDataSource(obj.dataSource);
  if (dsError) return dsError;

  const render = asObject(obj.render);
  if (!render) return "render must be an object.";
  if (render.kind !== "board") return 'render.kind must be "board".';

  if (typeof render.groupBy !== "string" || render.groupBy.trim() === "") {
    return "render.groupBy must be a non-empty string.";
  }
  const lanesError = validateLanes(render.lanes);
  if (lanesError) return lanesError;

  if (render.otherLane !== undefined && render.otherLane !== null && typeof render.otherLane !== "boolean") {
    return "render.otherLane must be a boolean.";
  }

  const cardError = validateCard(render.card);
  if (cardError) return cardError;

  return (
    validateBoardWhere(render.where) ??
    validateSort(render.sort) ??
    validateBoardRowActions(render.rowActions) ??
    optionalStringError("render.emptyMessage", render.emptyMessage)
  );
}

function validateLanes(raw: unknown): string | null {
  if (!Array.isArray(raw) || raw.length === 0) return "render.lanes must be a non-empty array.";
  for (let i = 0; i < raw.length; i += 1) {
    const lane = asObject(raw[i]);
    if (!lane) return `render.lanes[${i}] must be an object.`;
    if (typeof lane.value !== "string" || lane.value.trim() === "") {
      return `render.lanes[${i}].value must be a non-empty string.`;
    }
    const labelError = optionalStringError(`render.lanes[${i}].label`, lane.label);
    if (labelError) return labelError;
    if (lane.color !== undefined && lane.color !== null) {
      if (typeof lane.color !== "string" || !WIDGET_BOARD_LANE_COLORS.includes(lane.color as WidgetBoardLaneColor)) {
        return `render.lanes[${i}].color must be one of ${WIDGET_BOARD_LANE_COLORS.join(", ")}.`;
      }
    }
  }
  return null;
}

function validateCard(raw: unknown): string | null {
  const card = asObject(raw);
  if (!card) return "render.card must be an object.";
  if (typeof card.titleField !== "string" || card.titleField.trim() === "") {
    return "render.card.titleField must be a non-empty string.";
  }
  if (card.fields !== undefined && card.fields !== null) {
    if (!Array.isArray(card.fields)) return "render.card.fields must be an array.";
    for (let i = 0; i < card.fields.length; i += 1) {
      const field = asObject(card.fields[i]);
      if (!field) return `render.card.fields[${i}] must be an object.`;
      if (typeof field.field !== "string" || field.field.trim() === "") {
        return `render.card.fields[${i}].field must be a non-empty string.`;
      }
      const labelError = optionalStringError(`render.card.fields[${i}].label`, field.label);
      if (labelError) return labelError;
      const hrefError = optionalStringError(`render.card.fields[${i}].href`, field.href);
      if (hrefError) return hrefError;
      const showError = optionalStringError(`render.card.fields[${i}].show`, field.show);
      if (showError) return showError;
    }
  }
  return null;
}

function validateBoardWhere(where: unknown): string | null {
  if (where == null) return null;
  if (!Array.isArray(where)) return "render.where must be an array.";
  for (let i = 0; i < where.length; i += 1) {
    const item = asObject(where[i]);
    if (!item) return `render.where[${i}] must be an object.`;
    if (typeof item.field !== "string" || item.field.trim() === "") {
      return `render.where[${i}].field must be a non-empty string.`;
    }
    const op = item.op;
    if (typeof op !== "string" || !WIDGET_FILTER_OPS.includes(op as WidgetTableFilter["op"])) {
      return `render.where[${i}].op is not supported.`;
    }
  }
  return null;
}

function validateBoardRowActions(rowActions: unknown): string | null {
  if (rowActions == null) return null;
  if (!Array.isArray(rowActions)) return "render.rowActions must be an array.";
  for (let i = 0; i < rowActions.length; i += 1) {
    const action = normalizeRowAction(rowActions[i]);
    if (!action) return `render.rowActions[${i}] must be a trigger action.`;
    if (!action.node.trim()) return `render.rowActions[${i}].node must be set to a trigger node.`;
  }
  return null;
}
