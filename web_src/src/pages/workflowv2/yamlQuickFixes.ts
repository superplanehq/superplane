/**
 * Quick-fix (code action) provider for the canvas YAML editor.
 *
 * Each quick fix is registered as a `QuickFixHandler` keyed by its
 * `DiagnosticCode`. When Monaco asks for code actions on a marker that
 * carries a matching code, the handler is invoked to produce an edit.
 *
 * To add a new quick fix:
 *  1. Add a new DiagnosticCode in yamlCanvasValidation.ts.
 *  2. Tag the diagnostic with that code in the validation logic.
 *  3. Add a handler to QUICK_FIX_REGISTRY below.
 */

import type { Monaco } from "@monaco-editor/react";
import type { editor as MonacoEditor, languages as MonacoLanguages, IDisposable } from "monaco-editor";
import type { DiagnosticCode } from "./yamlCanvasValidation";

// ---------------------------------------------------------------------------
// Quick fix handler interface
// ---------------------------------------------------------------------------

interface QuickFixContext {
  monaco: Monaco;
  model: MonacoEditor.ITextModel;
  marker: MonacoEditor.IMarkerData;
  /** 1-based line where the marker starts (first line of the node block). */
  markerLine: number;
}

interface QuickFixResult {
  title: string;
  edits: MonacoLanguages.IWorkspaceTextEdit[];
}

type QuickFixHandler = (ctx: QuickFixContext) => QuickFixResult | null;

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

function generateRandomSuffix(): string {
  return Math.random().toString(36).substring(2, 8);
}

/**
 * Searches for a line matching `pattern` within the current node block,
 * starting from `startLine`. Stops at the next array item (`    - `).
 */
function findLineInNodeBlock(model: MonacoEditor.ITextModel, startLine: number, pattern: RegExp): number | null {
  const nextItemPattern = /^\s{4}- /;
  const lineCount = model.getLineCount();
  for (let i = startLine; i <= lineCount; i++) {
    const content = model.getLineContent(i);
    if (i > startLine && nextItemPattern.test(content)) break;
    if (pattern.test(content)) return i;
  }
  return null;
}

function makeTextEdit(
  monaco: Monaco,
  model: MonacoEditor.ITextModel,
  line: number,
  text: string,
): MonacoLanguages.IWorkspaceTextEdit {
  return {
    resource: model.uri,
    textEdit: {
      range: new monaco.Range(line, 1, line, model.getLineContent(line).length + 1),
      text,
    },
    versionId: model.getVersionId(),
  };
}

// ---------------------------------------------------------------------------
// Quick fix handlers
// ---------------------------------------------------------------------------

/** Generates a new unique ID, keeping the existing prefix. */
const fixDuplicateId: QuickFixHandler = ({ monaco, model, markerLine }) => {
  const idLine = findLineInNodeBlock(model, markerLine, /^\s+- id:\s*".*"/);
  if (!idLine) return null;

  const lineContent = model.getLineContent(idLine);
  const match = lineContent.match(/^(\s+- id:\s*")(.*)(")/);
  if (!match) return null;

  const oldId = match[2];
  const parts = oldId.split("-");
  const prefix = parts.length >= 2 ? parts.slice(0, -1).join("-") : oldId;
  const newId = `${prefix}-${generateRandomSuffix()}`;

  return {
    title: `Generate new ID: "${newId}"`,
    edits: [makeTextEdit(monaco, model, idLine, lineContent.replace(oldId, newId))],
  };
};

/** Shifts the node's position by (+500, +250) to resolve overlap. */
const fixPositionOverlap: QuickFixHandler = ({ monaco, model, markerLine }) => {
  const xLine = findLineInNodeBlock(model, markerLine, /^\s+x:\s/);
  const yLine = findLineInNodeBlock(model, markerLine, /^\s+"?y"?:\s/);
  if (!xLine || !yLine) return null;

  const xContent = model.getLineContent(xLine);
  const yContent = model.getLineContent(yLine);
  const xMatch = xContent.match(/^(\s+x:\s*)(\d+)/);
  const yMatch = yContent.match(/^(\s+"?y"?:\s*)(\d+)/);
  if (!xMatch || !yMatch) return null;

  const newX = parseInt(xMatch[2], 10) + 500;
  const newY = parseInt(yMatch[2], 10) + 250;

  return {
    title: `Shift position to (${newX}, ${newY})`,
    edits: [
      makeTextEdit(monaco, model, xLine, `${xMatch[1]}${newX}`),
      makeTextEdit(monaco, model, yLine, `${yMatch[1]}${newY}`),
    ],
  };
};

/** Appends a random suffix to the node name to resolve duplicate naming. */
const fixDuplicateName: QuickFixHandler = ({ monaco, model, markerLine }) => {
  const nameLine = findLineInNodeBlock(model, markerLine, /^\s+name:\s*"/);
  if (!nameLine) return null;

  const lineContent = model.getLineContent(nameLine);
  const match = lineContent.match(/^(\s+name:\s*")(.*)(")/);
  if (!match) return null;

  const oldName = match[2];
  const newName = `${oldName}-${generateRandomSuffix()}`;

  return {
    title: `Rename to "${newName}"`,
    edits: [makeTextEdit(monaco, model, nameLine, lineContent.replace(`"${oldName}"`, `"${newName}"`))],
  };
};

// ---------------------------------------------------------------------------
// Registry — add new quick fixes here
// ---------------------------------------------------------------------------

const QUICK_FIX_REGISTRY: Record<DiagnosticCode, QuickFixHandler> = {
  "duplicate-id": fixDuplicateId,
  "position-overlap": fixPositionOverlap,
  "duplicate-name": fixDuplicateName,
};

// ---------------------------------------------------------------------------
// Monaco code-action provider
// ---------------------------------------------------------------------------

export function registerQuickFixProvider(monaco: Monaco): IDisposable {
  return monaco.languages.registerCodeActionProvider("yaml", {
    provideCodeActions(model, _range, context) {
      const actions: MonacoLanguages.CodeAction[] = [];

      for (const marker of context.markers) {
        const code =
          typeof marker.code === "object" && marker.code !== null
            ? ((marker.code as { value: string }).value as DiagnosticCode)
            : undefined;

        if (!code) continue;

        const handler = QUICK_FIX_REGISTRY[code];
        if (!handler) continue;

        const result = handler({ monaco, model, marker, markerLine: marker.startLineNumber });
        if (!result) continue;

        actions.push({
          title: result.title,
          kind: "quickfix",
          diagnostics: [marker],
          isPreferred: true,
          edit: { edits: result.edits },
        });
      }

      return { actions, dispose() {} };
    },
  });
}
