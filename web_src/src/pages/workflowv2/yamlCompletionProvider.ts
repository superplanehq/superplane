/**
 * Structural YAML completion provider for the canvas editor.
 *
 * Provides context-aware suggestions based on cursor position:
 *  - Node/edge field names
 *  - Enum values (type, boolean fields)
 *  - Available component, trigger, and widget names
 *  - Node IDs for edge sourceId/targetId
 *  - Output channel names for edge channel
 *
 * To add new suggestions:
 *  1. Add the context detection logic in `getYamlContext`.
 *  2. Add the suggestion generation in `provideCompletionItems`.
 */

import type { Monaco } from "@monaco-editor/react";
import type { IDisposable, languages as MonacoLanguages } from "monaco-editor";
import type { ComponentsComponent, TriggersTrigger, WidgetsWidget } from "@/api-client/types.gen";

// ---------------------------------------------------------------------------
// Data passed from the component
// ---------------------------------------------------------------------------

export interface YamlCompletionData {
  components: ComponentsComponent[];
  triggers: TriggersTrigger[];
  widgets: WidgetsWidget[];
}

// ---------------------------------------------------------------------------
// YAML context detection
// ---------------------------------------------------------------------------

type YamlContext =
  | { kind: "node-field" }
  | { kind: "node-type-value" }
  | { kind: "bool-value" }
  | { kind: "component-name" }
  | { kind: "trigger-name" }
  | { kind: "widget-name" }
  | { kind: "edge-field" }
  | { kind: "edge-node-ref" }
  | { kind: "edge-channel"; sourceId: string | null }
  | { kind: "unknown" };

const NODE_FIELDS = [
  { name: "id", detail: "string — unique node identifier" },
  { name: "name", detail: "string — display name" },
  { name: "type", detail: "enum — TYPE_COMPONENT, TYPE_TRIGGER, etc." },
  { name: "configuration", detail: "object — node configuration" },
  { name: "metadata", detail: "object — node metadata" },
  { name: "position", detail: "object — { x, y } canvas position" },
  { name: "component", detail: "object — component reference" },
  { name: "blueprint", detail: "object — blueprint reference" },
  { name: "trigger", detail: "object — trigger reference" },
  { name: "widget", detail: "object — widget reference" },
  { name: "isCollapsed", detail: "bool — collapsed in canvas" },
  { name: "integration", detail: "object — integration reference" },
  { name: "paused", detail: "bool — paused state" },
];

const EDGE_FIELDS = [
  { name: "sourceId", detail: "string — source node ID" },
  { name: "targetId", detail: "string — target node ID" },
  { name: "channel", detail: "string — output channel name" },
];

const NODE_TYPE_VALUES = ["TYPE_COMPONENT", "TYPE_BLUEPRINT", "TYPE_TRIGGER", "TYPE_WIDGET"];

/**
 * Determines what kind of suggestion to offer based on the lines
 * above and at the cursor position.
 */
function getYamlContext(lines: string[], lineIndex: number, lineContent: string): YamlContext {
  const trimmed = lineContent.trimStart();

  // Detect if we're inside the edges section
  const inEdges = isInSection(lines, lineIndex, "edges");
  const inNodes = isInSection(lines, lineIndex, "nodes");

  // After "type:" on a node line → suggest enum values
  if (inNodes && /^\s+type:\s*"?/.test(lineContent)) {
    return { kind: "node-type-value" };
  }

  // After "isCollapsed:" or "paused:" → suggest boolean
  if (inNodes && /^\s+(isCollapsed|paused):\s*/.test(lineContent)) {
    return { kind: "bool-value" };
  }

  // Inside a component ref block → suggest component names
  if (inNodes && /^\s+name:\s*"?/.test(lineContent)) {
    const parentRef = findParentRefKey(lines, lineIndex);
    if (parentRef === "component") return { kind: "component-name" };
    if (parentRef === "trigger") return { kind: "trigger-name" };
    if (parentRef === "widget") return { kind: "widget-name" };
  }

  // Edge section: after sourceId/targetId → suggest node IDs
  if (inEdges && /^\s+(sourceId|targetId):\s*"?/.test(lineContent)) {
    return { kind: "edge-node-ref" };
  }

  // Edge section: after channel → suggest channel names
  if (inEdges && /^\s+channel:\s*"?/.test(lineContent)) {
    const sourceId = findEdgeSourceId(lines, lineIndex);
    return { kind: "edge-channel", sourceId };
  }

  // At the start of a new field in an edge block
  if (inEdges && (trimmed === "" || trimmed === "- " || /^\s{6}\w*$/.test(lineContent))) {
    return { kind: "edge-field" };
  }

  // At the start of a new field in a node block
  if (inNodes && (trimmed === "" || /^\s{6}\w*$/.test(lineContent))) {
    return { kind: "node-field" };
  }

  return { kind: "unknown" };
}

function isInSection(lines: string[], lineIndex: number, section: "nodes" | "edges"): boolean {
  const sectionPattern = new RegExp(`^\\s+${section}:\\s*$`);
  const otherSection = section === "nodes" ? "edges" : "nodes";
  const otherPattern = new RegExp(`^\\s+${otherSection}:\\s*$`);

  for (let i = lineIndex; i >= 0; i--) {
    if (sectionPattern.test(lines[i])) return true;
    if (otherPattern.test(lines[i])) return false;
    if (/^[a-zA-Z]/.test(lines[i]) && i < lineIndex) return false;
  }
  return false;
}

/** Walks up from a `name:` line to find if the parent key is component/trigger/widget. */
function findParentRefKey(lines: string[], lineIndex: number): string | null {
  const nameIndent = lines[lineIndex].search(/\S/);
  for (let i = lineIndex - 1; i >= 0; i--) {
    const indent = lines[i].search(/\S/);
    if (indent < nameIndent && indent >= 0) {
      const match = lines[i].match(/^\s+(component|trigger|widget):/);
      if (match) return match[1];
      return null;
    }
  }
  return null;
}

/** Finds the sourceId value for the current edge block. */
function findEdgeSourceId(lines: string[], lineIndex: number): string | null {
  const arrayItemPattern = /^\s{4}- /;
  for (let i = lineIndex; i >= 0; i--) {
    const sourceMatch = lines[i].match(/^\s+(?:- )?sourceId:\s*"([^"]*)"/);
    if (sourceMatch) return sourceMatch[1];
    if (i < lineIndex && arrayItemPattern.test(lines[i])) break;
  }
  return null;
}

/** Extracts all node IDs from the YAML text. */
function extractNodeIds(fullText: string): string[] {
  const ids: string[] = [];
  const pattern = /^\s+- id:\s*"([^"]*)"/gm;
  let match;
  while ((match = pattern.exec(fullText)) !== null) {
    ids.push(match[1]);
  }
  return ids;
}

/** Extracts the component name for a given node sourceId. */
function findComponentNameForNode(fullText: string, nodeId: string): string | null {
  const lines = fullText.split("\n");
  let found = false;
  for (const line of lines) {
    if (/^\s+- id:/.test(line)) {
      found = line.includes(`"${nodeId}"`);
    }
    if (found) {
      const compMatch = line.match(/^\s+name:\s*"([^"]*)"/);
      if (compMatch) {
        const prevLines = lines.slice(0, lines.indexOf(line));
        for (let j = prevLines.length - 1; j >= 0; j--) {
          if (/^\s+component:/.test(prevLines[j])) return compMatch[1];
          if (/^\s+- id:/.test(prevLines[j])) break;
        }
      }
    }
  }
  return null;
}

// ---------------------------------------------------------------------------
// Provider registration
// ---------------------------------------------------------------------------

const dataRef: { current: YamlCompletionData } = {
  current: { components: [], triggers: [], widgets: [] },
};

export function updateYamlCompletionData(data: YamlCompletionData): void {
  dataRef.current = data;
}

export function registerYamlCompletionProvider(monaco: Monaco): IDisposable {
  return monaco.languages.registerCompletionItemProvider("yaml", {
    triggerCharacters: [":", " ", '"', "\n"],

    provideCompletionItems(model, position) {
      const fullText = model.getValue();
      const lines = fullText.split("\n");
      const lineIndex = position.lineNumber - 1;
      const lineContent = lines[lineIndex] ?? "";

      const ctx = getYamlContext(lines, lineIndex, lineContent);
      const word = model.getWordUntilPosition(position);
      const range = new monaco.Range(position.lineNumber, word.startColumn, position.lineNumber, word.endColumn);

      const suggestions: MonacoLanguages.CompletionItem[] = [];
      const data = dataRef.current;

      switch (ctx.kind) {
        case "node-field":
          for (const field of NODE_FIELDS) {
            suggestions.push({
              label: field.name,
              kind: monaco.languages.CompletionItemKind.Field,
              insertText: `${field.name}: `,
              detail: field.detail,
              range,
            });
          }
          break;

        case "node-type-value":
          for (const val of NODE_TYPE_VALUES) {
            suggestions.push({
              label: val,
              kind: monaco.languages.CompletionItemKind.Enum,
              insertText: `"${val}"`,
              range,
            });
          }
          break;

        case "bool-value":
          for (const val of ["true", "false"]) {
            suggestions.push({
              label: val,
              kind: monaco.languages.CompletionItemKind.Value,
              insertText: val,
              range,
            });
          }
          break;

        case "component-name":
          for (const comp of data.components) {
            if (!comp.name) continue;
            suggestions.push({
              label: comp.name,
              kind: monaco.languages.CompletionItemKind.Module,
              insertText: `"${comp.name}"`,
              detail: comp.label || comp.description || "Component",
              range,
            });
          }
          break;

        case "trigger-name":
          for (const trig of data.triggers) {
            if (!trig.name) continue;
            suggestions.push({
              label: trig.name,
              kind: monaco.languages.CompletionItemKind.Module,
              insertText: `"${trig.name}"`,
              detail: trig.label || trig.description || "Trigger",
              range,
            });
          }
          break;

        case "widget-name":
          for (const w of data.widgets) {
            if (!w.name) continue;
            suggestions.push({
              label: w.name,
              kind: monaco.languages.CompletionItemKind.Module,
              insertText: `"${w.name}"`,
              detail: w.label || w.description || "Widget",
              range,
            });
          }
          break;

        case "edge-field":
          for (const field of EDGE_FIELDS) {
            suggestions.push({
              label: field.name,
              kind: monaco.languages.CompletionItemKind.Field,
              insertText: `${field.name}: `,
              detail: field.detail,
              range,
            });
          }
          break;

        case "edge-node-ref": {
          const nodeIds = extractNodeIds(fullText);
          for (const id of nodeIds) {
            suggestions.push({
              label: id,
              kind: monaco.languages.CompletionItemKind.Reference,
              insertText: `"${id}"`,
              range,
            });
          }
          break;
        }

        case "edge-channel": {
          suggestions.push({
            label: "default",
            kind: monaco.languages.CompletionItemKind.Value,
            insertText: '"default"',
            detail: "Default output channel",
            range,
          });
          if (ctx.sourceId) {
            const compName = findComponentNameForNode(fullText, ctx.sourceId);
            if (compName) {
              const comp = data.components.find((c) => c.name === compName);
              if (comp?.outputChannels) {
                for (const ch of comp.outputChannels) {
                  if (ch.name && ch.name !== "default") {
                    suggestions.push({
                      label: ch.name,
                      kind: monaco.languages.CompletionItemKind.Value,
                      insertText: `"${ch.name}"`,
                      detail: ch.label || ch.description || "Output channel",
                      range,
                    });
                  }
                }
              }
            }
          }
          break;
        }
      }

      return { suggestions };
    },
  });
}
