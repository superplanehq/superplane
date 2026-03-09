/**
 * Client-side validation for canvas YAML.
 *
 * Mirrors key backend validation rules (see pkg/grpc/actions/canvases/serialization.go)
 * so that errors surface inline in the Monaco editor before a save round-trip.
 *
 * Architecture:
 *  - `validateCanvasYaml` is the single entry point. It returns an array of
 *    `YamlDiagnostic` objects that the editor maps to markers (underlines)
 *    and decorations (block highlights).
 *  - Each diagnostic has a `kind`:
 *      "line"  → squiggly underline on a single line
 *      "block" → background highlight spanning the full node block
 *  - Diagnostics with `severity: "error"` block saves; "warning" does not.
 *  - An optional `code` field (DiagnosticCode) tags fixable diagnostics so
 *    the code-action provider in yamlQuickFixes.ts can offer quick fixes.
 *
 * To add a new validation rule:
 *  1. Add the check inside `validateCanvasYaml` (or a helper it calls).
 *  2. If the rule should be fixable, add a new DiagnosticCode and tag the
 *     diagnostic, then register a matching quick fix in yamlQuickFixes.ts.
 */

// ---------------------------------------------------------------------------
// Diagnostic types
// ---------------------------------------------------------------------------

/** Codes that identify fixable diagnostics. Add new codes here when adding quick fixes. */
export type DiagnosticCode = "duplicate-id" | "position-overlap" | "duplicate-name";

export interface YamlDiagnostic {
  startLineNumber: number;
  endLineNumber: number;
  startColumn: number;
  endColumn: number;
  message: string;
  severity: "error" | "warning";
  /** "line" = single-line underline, "block" = full node-block background highlight */
  kind: "line" | "block";
  /** When set, the code-action provider can offer a quick fix for this diagnostic */
  code?: DiagnosticCode;
}

// ---------------------------------------------------------------------------
// Parsed YAML shapes (mirrors the proto definitions in protos/components.proto)
// ---------------------------------------------------------------------------

interface ParsedNode {
  id?: string;
  name?: string;
  type?: string;
  configuration?: unknown;
  metadata?: unknown;
  position?: { x?: unknown; y?: unknown } | null;
  component?: { name?: string } | null;
  blueprint?: { id?: string } | null;
  trigger?: { name?: string } | null;
  widget?: { name?: string } | null;
  isCollapsed?: unknown;
  integration?: { id?: string; name?: string } | null;
  errorMessage?: string;
  warningMessage?: string;
  paused?: unknown;
}

interface ParsedEdge {
  sourceId?: string;
  targetId?: string;
  channel?: unknown;
}

interface ParsedCanvas {
  metadata?: { name?: string };
  spec?: {
    nodes?: ParsedNode[];
    edges?: ParsedEdge[];
  };
}

// ---------------------------------------------------------------------------
// Field-type rules (proto type → JS typeof check)
// ---------------------------------------------------------------------------

type FieldTypeRule = { field: string; type: "bool" | "string" | "integer" | "object" };

/** Top-level fields on a Node and their expected proto types. */
const NODE_FIELD_TYPES: FieldTypeRule[] = [
  { field: "id", type: "string" },
  { field: "name", type: "string" },
  { field: "type", type: "string" },
  { field: "isCollapsed", type: "bool" },
  { field: "paused", type: "bool" },
  { field: "errorMessage", type: "string" },
  { field: "warningMessage", type: "string" },
  { field: "configuration", type: "object" },
  { field: "metadata", type: "object" },
  { field: "position", type: "object" },
  { field: "component", type: "object" },
  { field: "blueprint", type: "object" },
  { field: "trigger", type: "object" },
  { field: "widget", type: "object" },
  { field: "integration", type: "object" },
];

const POSITION_FIELD_TYPES: FieldTypeRule[] = [
  { field: "x", type: "integer" },
  { field: "y", type: "integer" },
];

function checkFieldType(value: unknown, rule: FieldTypeRule): boolean {
  if (value === null || value === undefined) return true;
  switch (rule.type) {
    case "bool":
      return typeof value === "boolean";
    case "string":
      return typeof value === "string";
    case "integer":
      return typeof value === "number" && Number.isInteger(value);
    case "object":
      return typeof value === "object";
  }
}

// ---------------------------------------------------------------------------
// Line-range helpers — map parsed array indices back to YAML line numbers
// ---------------------------------------------------------------------------

/** Inclusive 1-based line range within the YAML text. */
interface LineRange {
  start: number;
  end: number;
}

/**
 * Returns the line range for each node in the `spec.nodes` array.
 * Ranges are bounded by the next array item or the `edges:` header.
 */
function findNodeLineRanges(lines: string[]): LineRange[] {
  const ranges: LineRange[] = [];
  const nodesHeaderPattern = /^\s+nodes:\s*$/;
  const edgesHeaderPattern = /^\s+edges:\s*$/;
  const arrayItemPattern = /^\s{4}- /;
  let inNodes = false;

  for (let i = 0; i < lines.length; i++) {
    if (nodesHeaderPattern.test(lines[i])) {
      inNodes = true;
      continue;
    }
    if (inNodes && edgesHeaderPattern.test(lines[i])) {
      if (ranges.length > 0) {
        ranges[ranges.length - 1].end = i;
      }
      break;
    }
    if (inNodes && arrayItemPattern.test(lines[i])) {
      if (ranges.length > 0) {
        ranges[ranges.length - 1].end = i;
      }
      ranges.push({ start: i + 1, end: lines.length });
    }
  }

  return ranges;
}

/** Returns the 1-based line number of the Nth edge array item. */
function findEdgeLine(lines: string[], edgeIndex: number): number {
  let arrayItemIndex = -1;
  const edgesHeaderPattern = /^\s+edges:\s*$/;
  const arrayItemPattern = /^\s{4}- /;
  let inEdges = false;

  for (let i = 0; i < lines.length; i++) {
    if (edgesHeaderPattern.test(lines[i])) {
      inEdges = true;
      continue;
    }
    if (inEdges && arrayItemPattern.test(lines[i])) {
      arrayItemIndex++;
      if (arrayItemIndex === edgeIndex) {
        return i + 1;
      }
    }
  }
  return 1;
}

/** Finds the 1-based line of a YAML field within a node's line range. */
function findFieldLine(lines: string[], range: LineRange | undefined, fieldName: string): number | null {
  if (!range) return null;
  const pattern = new RegExp(`^\\s+${fieldName}:\\s`);
  for (let i = range.start - 1; i < range.end; i++) {
    if (pattern.test(lines[i])) {
      return i + 1;
    }
  }
  return null;
}

// ---------------------------------------------------------------------------
// Diagnostic factory helpers
// ---------------------------------------------------------------------------

/** Single-line diagnostic (squiggly underline). */
function makeDiagnostic(
  line: number,
  lineText: string,
  message: string,
  severity: "error" | "warning",
): YamlDiagnostic {
  return {
    startLineNumber: line,
    endLineNumber: line,
    startColumn: 1,
    endColumn: (lineText?.length ?? 0) + 1,
    message,
    severity,
    kind: "line",
  };
}

/** Block diagnostic (background highlight spanning the full node). */
function makeRangeDiagnostic(
  range: LineRange,
  message: string,
  severity: "error" | "warning",
  code?: DiagnosticCode,
): YamlDiagnostic {
  return {
    startLineNumber: range.start,
    endLineNumber: range.end,
    startColumn: 1,
    endColumn: 1,
    message,
    severity,
    kind: "block",
    code,
  };
}

/** Block diagnostic when range is available, falls back to single-line. */
function makeNodeDiagnostic(
  range: LineRange | undefined,
  line: number,
  lineText: string,
  message: string,
  severity: "error" | "warning",
  code?: DiagnosticCode,
): YamlDiagnostic {
  if (range) {
    return makeRangeDiagnostic(range, message, severity, code);
  }
  return makeDiagnostic(line, lineText, message, severity);
}

/**
 * Emits both a block highlight AND a line underline on the specific field.
 * Used for type-mismatch errors where we want to point at the exact field
 * but also light up the whole node.
 */
function addFieldTypeDiagnostic(
  range: LineRange | undefined,
  lines: string[],
  fieldName: string,
  message: string,
  diagnostics: YamlDiagnostic[],
): void {
  if (range) {
    diagnostics.push(makeRangeDiagnostic(range, message, "error"));
  }
  const fieldLine = findFieldLine(lines, range, fieldName);
  const line = fieldLine ?? range?.start ?? 1;
  const lt = lines[line - 1] ?? "";
  diagnostics.push(makeDiagnostic(line, lt, message, "error"));
}

// ---------------------------------------------------------------------------
// Validation entry point
// ---------------------------------------------------------------------------

export function validateCanvasYaml(parsed: Record<string, unknown>, yamlText: string): YamlDiagnostic[] {
  const canvas = parsed as ParsedCanvas;
  const diagnostics: YamlDiagnostic[] = [];
  const lines = yamlText.split("\n");

  // -- Canvas-level checks --------------------------------------------------

  if (!canvas.metadata?.name) {
    diagnostics.push(makeDiagnostic(1, lines[0], "Canvas name is required", "error"));
  }

  const nodes = canvas.spec?.nodes;
  if (!nodes || !Array.isArray(nodes)) {
    return diagnostics;
  }

  // -- Node-level checks ----------------------------------------------------

  const nodeRanges = findNodeLineRanges(lines);
  const seenIds = new Map<string, number>();

  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    const range = nodeRanges[i];
    const line = range?.start ?? 1;
    const lineText = lines[line - 1] ?? "";

    if (!node.id) {
      diagnostics.push(makeNodeDiagnostic(range, line, lineText, `Node ${i}: id is required`, "error"));
      continue;
    }

    if (!node.name) {
      diagnostics.push(makeNodeDiagnostic(range, line, lineText, `Node "${node.id}": name is required`, "error"));
    }

    if (seenIds.has(node.id)) {
      diagnostics.push(
        makeNodeDiagnostic(range, line, lineText, `Node "${node.id}": duplicate node id`, "error", "duplicate-id"),
      );
    } else {
      seenIds.set(node.id, i);
    }

    if (!node.type) {
      diagnostics.push(makeNodeDiagnostic(range, line, lineText, `Node "${node.id}": type is required`, "error"));
    } else {
      validateNodeRef(node, range, line, lineText, diagnostics);
    }

    validateNodeFieldTypes(
      nodes[i] as unknown as Record<string, unknown>,
      range,
      lines,
      `Node "${node.id}"`,
      diagnostics,
    );

    // Surface server-side errorMessage / warningMessage as block highlights.
    // These are informational — they don't block saves.
    if (range && node.errorMessage) {
      diagnostics.push(makeRangeDiagnostic(range, `Node "${node.id}": ${node.errorMessage}`, "warning"));
    }

    if (range && node.warningMessage) {
      const warnCode: DiagnosticCode | undefined = /multiple\s+(components|triggers)\s+named/i.test(node.warningMessage)
        ? "duplicate-name"
        : undefined;
      diagnostics.push(makeRangeDiagnostic(range, `Node "${node.id}": ${node.warningMessage}`, "warning", warnCode));
    }
  }

  // -- Position overlap check (warning, does not block saves) ---------------

  const seenPositions = new Map<string, { id: string; index: number }>();
  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    if (!node.id || !node.position) continue;
    const x = node.position.x;
    const y = node.position.y;
    if (typeof x !== "number" || typeof y !== "number") continue;

    const key = `${x},${y}`;
    const existing = seenPositions.get(key);
    if (existing) {
      const range = nodeRanges[i];
      if (range) {
        diagnostics.push(
          makeRangeDiagnostic(
            range,
            `Node "${node.id}" overlaps with node "${existing.id}" at position (${x}, ${y})`,
            "warning",
            "position-overlap",
          ),
        );
      }
      const existingRange = nodeRanges[existing.index];
      if (existingRange) {
        diagnostics.push(
          makeRangeDiagnostic(
            existingRange,
            `Node "${existing.id}" overlaps with node "${node.id}" at position (${x}, ${y})`,
            "warning",
            "position-overlap",
          ),
        );
      }
    } else {
      seenPositions.set(key, { id: node.id, index: i });
    }
  }

  // -- Edge-level checks ----------------------------------------------------

  const edges = canvas.spec?.edges;
  if (!edges || !Array.isArray(edges)) {
    return diagnostics;
  }

  const nodeIds = new Set(nodes.filter((n) => n.id).map((n) => n.id!));

  for (let i = 0; i < edges.length; i++) {
    const edge = edges[i];
    const line = findEdgeLine(lines, i);
    const lineText = lines[line - 1] ?? "";

    if (!edge.sourceId || !edge.targetId) {
      diagnostics.push(makeDiagnostic(line, lineText, `Edge ${i}: sourceId and targetId are required`, "error"));
      continue;
    }

    if (!nodeIds.has(edge.sourceId)) {
      diagnostics.push(makeDiagnostic(line, lineText, `Edge ${i}: source node "${edge.sourceId}" not found`, "error"));
    }

    if (!nodeIds.has(edge.targetId)) {
      diagnostics.push(makeDiagnostic(line, lineText, `Edge ${i}: target node "${edge.targetId}" not found`, "error"));
    }
  }

  return diagnostics;
}

// ---------------------------------------------------------------------------
// Per-node sub-validators
// ---------------------------------------------------------------------------

/** Validates that scalar fields on a node match their expected proto types. */
function validateNodeFieldTypes(
  node: Record<string, unknown>,
  range: LineRange | undefined,
  lines: string[],
  nodeLabel: string,
  diagnostics: YamlDiagnostic[],
): void {
  for (const rule of NODE_FIELD_TYPES) {
    const value = node[rule.field];
    if (!checkFieldType(value, rule)) {
      addFieldTypeDiagnostic(
        range,
        lines,
        rule.field,
        `${nodeLabel}: "${rule.field}" must be a ${rule.type}, got ${typeof value}`,
        diagnostics,
      );
    }
  }

  const pos = node.position as { x?: unknown; y?: unknown } | null | undefined;
  if (pos && typeof pos === "object") {
    for (const rule of POSITION_FIELD_TYPES) {
      const value = (pos as Record<string, unknown>)[rule.field];
      if (!checkFieldType(value, rule)) {
        addFieldTypeDiagnostic(
          range,
          lines,
          rule.field,
          `${nodeLabel}: position.${rule.field} must be an ${rule.type}, got ${typeof value}`,
          diagnostics,
        );
      }
    }
  }
}

/** Validates that the node has the correct ref field for its type (component, trigger, etc.). */
function validateNodeRef(
  node: ParsedNode,
  range: LineRange | undefined,
  line: number,
  lineText: string,
  diagnostics: YamlDiagnostic[],
): void {
  const id = node.id ?? "?";
  switch (node.type) {
    case "TYPE_COMPONENT":
      if (!node.component) {
        diagnostics.push(
          makeNodeDiagnostic(range, line, lineText, `Node "${id}": component reference is required`, "error"),
        );
      } else if (!node.component.name) {
        diagnostics.push(
          makeNodeDiagnostic(range, line, lineText, `Node "${id}": component name is required`, "error"),
        );
      }
      break;
    case "TYPE_TRIGGER":
      if (!node.trigger) {
        diagnostics.push(
          makeNodeDiagnostic(range, line, lineText, `Node "${id}": trigger reference is required`, "error"),
        );
      } else if (!node.trigger.name) {
        diagnostics.push(makeNodeDiagnostic(range, line, lineText, `Node "${id}": trigger name is required`, "error"));
      }
      break;
    case "TYPE_BLUEPRINT":
      if (!node.blueprint) {
        diagnostics.push(
          makeNodeDiagnostic(range, line, lineText, `Node "${id}": blueprint reference is required`, "error"),
        );
      } else if (!node.blueprint.id) {
        diagnostics.push(makeNodeDiagnostic(range, line, lineText, `Node "${id}": blueprint ID is required`, "error"));
      }
      break;
    case "TYPE_WIDGET":
      if (!node.widget) {
        diagnostics.push(
          makeNodeDiagnostic(range, line, lineText, `Node "${id}": widget reference is required`, "error"),
        );
      } else if (!node.widget.name) {
        diagnostics.push(makeNodeDiagnostic(range, line, lineText, `Node "${id}": widget name is required`, "error"));
      }
      break;
  }
}
