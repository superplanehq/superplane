/**
 * Adapters between the SDK panel-type enum and the internal FE `PanelType`
 * union.
 *
 * The generated TS SDK serializes `Console.Panel.type` as the SCREAMING_CASE
 * proto enum string (e.g. `"MARKDOWN"`). The internal FE schema is lowercase
 * (`"markdown"`) so it matches the YAML contract and the existing per-type
 * validators in `panelTypes.ts`. Anyone adding a new panel kind has a single
 * place to keep all three mirrors in sync: the proto enum, the lowercase FE
 * union in `panelTypes.ts`, and these mapping tables.
 *
 * These lived in `panelTypes.ts` originally but were extracted to keep that
 * file under the `max-lines` lint budget.
 */

import type { ConsolePanelType as ApiPanelType } from "@/api-client";

import { type PanelType } from "./panelTypes";

const API_TO_PANEL_TYPE: Record<Exclude<ApiPanelType, "TYPE_UNSPECIFIED">, PanelType> = {
  MARKDOWN: "markdown",
  NODE: "node",
  NODES: "nodes",
  TABLE: "table",
  CHART: "chart",
  NUMBER: "number",
};

const PANEL_TYPE_TO_API: Record<PanelType, ApiPanelType> = {
  markdown: "MARKDOWN",
  node: "NODE",
  nodes: "NODES",
  table: "TABLE",
  chart: "CHART",
  number: "NUMBER",
};

/**
 * Convert the SDK enum value to the internal lowercase form. Returns
 * `undefined` for `TYPE_UNSPECIFIED` / unknown values so the caller can
 * decide between dropping the panel and falling back to a default.
 */
export function apiPanelTypeToPanelType(value: ApiPanelType | string | undefined | null): PanelType | undefined {
  if (!value || value === "TYPE_UNSPECIFIED") return undefined;
  return API_TO_PANEL_TYPE[value as Exclude<ApiPanelType, "TYPE_UNSPECIFIED">];
}

/**
 * Convert the internal lowercase form to the SDK enum value used on the
 * wire. Total over `PanelType` by construction.
 */
export function panelTypeToApi(value: PanelType): ApiPanelType {
  return PANEL_TYPE_TO_API[value];
}
