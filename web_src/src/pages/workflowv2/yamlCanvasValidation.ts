export interface YamlDiagnostic {
  startLineNumber: number;
  endLineNumber: number;
  startColumn: number;
  endColumn: number;
  message: string;
  severity: "error" | "warning";
}

interface ParsedNode {
  id?: string;
  name?: string;
  type?: string;
  component?: { name?: string } | null;
  blueprint?: { id?: string } | null;
  trigger?: { name?: string } | null;
  widget?: { name?: string } | null;
}

interface ParsedEdge {
  sourceId?: string;
  targetId?: string;
}

interface ParsedCanvas {
  metadata?: { name?: string };
  spec?: {
    nodes?: ParsedNode[];
    edges?: ParsedEdge[];
  };
}

function findNodeLine(lines: string[], nodeIndex: number): number {
  let arrayItemIndex = -1;
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
      break;
    }
    if (inNodes && arrayItemPattern.test(lines[i])) {
      arrayItemIndex++;
      if (arrayItemIndex === nodeIndex) {
        return i + 1;
      }
    }
  }
  return 1;
}

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
  };
}

export function validateCanvasYaml(parsed: Record<string, unknown>, yamlText: string): YamlDiagnostic[] {
  const canvas = parsed as ParsedCanvas;
  const diagnostics: YamlDiagnostic[] = [];
  const lines = yamlText.split("\n");

  if (!canvas.metadata?.name) {
    diagnostics.push(makeDiagnostic(1, lines[0], "Canvas name is required", "error"));
  }

  const nodes = canvas.spec?.nodes;
  if (!nodes || !Array.isArray(nodes)) {
    return diagnostics;
  }

  const seenIds = new Map<string, number>();

  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    const line = findNodeLine(lines, i);
    const lineText = lines[line - 1] ?? "";

    if (!node.id) {
      diagnostics.push(makeDiagnostic(line, lineText, `Node ${i}: id is required`, "error"));
      continue;
    }

    if (!node.name) {
      diagnostics.push(makeDiagnostic(line, lineText, `Node "${node.id}": name is required`, "error"));
    }

    if (seenIds.has(node.id)) {
      diagnostics.push(makeDiagnostic(line, lineText, `Node "${node.id}": duplicate node id`, "error"));
    } else {
      seenIds.set(node.id, i);
    }

    if (!node.type) {
      diagnostics.push(makeDiagnostic(line, lineText, `Node "${node.id}": type is required`, "error"));
    } else {
      validateNodeRef(node, line, lineText, diagnostics);
    }
  }

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

function validateNodeRef(node: ParsedNode, line: number, lineText: string, diagnostics: YamlDiagnostic[]): void {
  const id = node.id ?? "?";
  switch (node.type) {
    case "TYPE_COMPONENT":
      if (!node.component) {
        diagnostics.push(makeDiagnostic(line, lineText, `Node "${id}": component reference is required`, "error"));
      } else if (!node.component.name) {
        diagnostics.push(makeDiagnostic(line, lineText, `Node "${id}": component name is required`, "error"));
      }
      break;
    case "TYPE_TRIGGER":
      if (!node.trigger) {
        diagnostics.push(makeDiagnostic(line, lineText, `Node "${id}": trigger reference is required`, "error"));
      } else if (!node.trigger.name) {
        diagnostics.push(makeDiagnostic(line, lineText, `Node "${id}": trigger name is required`, "error"));
      }
      break;
    case "TYPE_BLUEPRINT":
      if (!node.blueprint) {
        diagnostics.push(makeDiagnostic(line, lineText, `Node "${id}": blueprint reference is required`, "error"));
      } else if (!node.blueprint.id) {
        diagnostics.push(makeDiagnostic(line, lineText, `Node "${id}": blueprint ID is required`, "error"));
      }
      break;
    case "TYPE_WIDGET":
      if (!node.widget) {
        diagnostics.push(makeDiagnostic(line, lineText, `Node "${id}": widget reference is required`, "error"));
      } else if (!node.widget.name) {
        diagnostics.push(makeDiagnostic(line, lineText, `Node "${id}": widget name is required`, "error"));
      }
      break;
  }
}
