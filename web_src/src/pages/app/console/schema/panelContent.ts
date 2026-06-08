/**
 * Source-of-truth TypeScript union for the JSON Schema that documents the
 * `Console.Panel.content` shape on the OpenAPI surface.
 *
 * The runtime FE types in `../panelTypes.ts` and `../nodesPanelContent.ts`
 * are the authoritative shapes the editor + validators use. This module just
 * collects them into a union so a schema generator (currently
 * `ts-json-schema-generator`) can emit a single JSON Schema that gets injected
 * into the generated `superplane.swagger.json` by a post-processing step (see
 * `Makefile` + `scripts/inject-console-panel-content-schema.mjs`).
 *
 * IMPORTANT: this union models the **content payload only**, i.e. exactly what
 * travels in `Console.Panel.content` on the wire (a `google.protobuf.Value`).
 * The discriminator (`Console.Panel.type`) is a sibling field on the panel, not
 * part of the content, so it must NOT be wrapped here. Wrapping each variant as
 * `{ type, content }` would describe the whole panel object and cause every
 * valid `content` payload (e.g. a markdown `{ body }`) to fail validation.
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
  | MarkdownPanelContent
  | NodePanelContent
  | NodesPanelContent
  | TablePanelContent
  | ChartPanelContent
  | NumberPanelContent;
