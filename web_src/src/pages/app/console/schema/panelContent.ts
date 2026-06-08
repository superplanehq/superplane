/**
 * Source-of-truth TypeScript union for the JSON Schema that documents the
 * `Console.Panel.content` shape on the OpenAPI surface.
 *
 * The runtime FE types in `../panelTypes.ts` and `../nodesPanelContent.ts`
 * are the authoritative shapes the editor + validators use. This module just
 * collects them into a discriminated union keyed by panel `type` so a
 * schema generator (currently `ts-json-schema-generator`) can emit a single
 * JSON Schema that gets injected into the generated `superplane.swagger.json`
 * by a post-processing step (see `Makefile` + `scripts/inject-console-panel-content-schema.mjs`).
 *
 * Keep this union exhaustive over `PanelType`: adding a new panel kind must
 * extend this union in the same change that adds its content interface.
 */

import type {
  ChartPanelContent,
  MarkdownPanelContent,
  NodePanelContent,
  NumberPanelContent,
  TablePanelContent,
} from "../panelTypes";
import type { NodesPanelContent } from "../nodesPanelContent";

export type ConsolePanelContent =
  | { type: "markdown"; content: MarkdownPanelContent }
  | { type: "node"; content: NodePanelContent }
  | { type: "nodes"; content: NodesPanelContent }
  | { type: "table"; content: TablePanelContent }
  | { type: "chart"; content: ChartPanelContent }
  | { type: "number"; content: NumberPanelContent };
