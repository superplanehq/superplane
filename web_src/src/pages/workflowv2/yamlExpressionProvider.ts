/**
 * Expression autocomplete provider for the canvas YAML editor.
 *
 * Activates inside `{{ }}` blocks within YAML string values, reusing the
 * same `getSuggestions` engine from AutoCompleteInput/core.ts that powers
 * expression autocomplete in the component sidebar field renderers.
 *
 * The provider detects `{{ }}` delimiters in the current line, extracts the
 * expression text and cursor offset, then delegates to `getSuggestions`.
 */

import type { Monaco } from "@monaco-editor/react";
import type { IDisposable, languages as MonacoLanguages } from "monaco-editor";
import { getSuggestions, type Suggestion } from "@/components/AutoCompleteInput/core";

// ---------------------------------------------------------------------------
// Expression context shared state
// ---------------------------------------------------------------------------

const exampleObjRef: { current: Record<string, unknown> | null } = { current: null };

export function updateYamlExpressionData(exampleObj: Record<string, unknown> | null): void {
  exampleObjRef.current = exampleObj;
}

// ---------------------------------------------------------------------------
// Helpers (adapted from useMonacoExpressionAutocomplete.ts)
// ---------------------------------------------------------------------------

const START_WORD = "{{";
const SUFFIX = "}}";

const SUGGESTION_SORT_PRIORITY: Record<string, number> = { $: 1, root: 2, previous: 3 };

function getExpressionContext(text: string, cursor: number) {
  const openIndex = text.lastIndexOf(START_WORD, cursor);
  if (openIndex === -1) return null;

  const closeIndex = text.indexOf(SUFFIX, openIndex + START_WORD.length);
  if (closeIndex !== -1 && cursor > closeIndex) return null;

  const startOffset = openIndex + START_WORD.length;
  const endOffset = closeIndex === -1 ? text.length : closeIndex;
  return {
    expressionText: text.slice(startOffset, endOffset),
    expressionCursor: Math.max(0, cursor - startOffset),
    startOffset,
    endOffset,
  };
}

function getCompletionKind(monaco: Monaco, kind: Suggestion["kind"]): MonacoLanguages.CompletionItemKind {
  switch (kind) {
    case "function":
      return monaco.languages.CompletionItemKind.Function;
    case "field":
      return monaco.languages.CompletionItemKind.Field;
    case "keyword":
      return monaco.languages.CompletionItemKind.Keyword;
    case "variable":
    default:
      return monaco.languages.CompletionItemKind.Variable;
  }
}

function getInsertText(suggestion: Suggestion): string {
  if (suggestion.kind === "function") {
    return suggestion.insertText ?? `${suggestion.label}()`;
  }
  return suggestion.insertText ?? suggestion.label;
}

function normalizeBracketQuotes(insertText: string): string {
  const normalizeInner = (value: string) => value.replace(/\\"/g, '"').replace(/'/g, "\\'");
  let next = insertText;
  next = next.replace(/\$\["((?:\\.|[^"\\])*)"\]/g, (_, inner) => `$['${normalizeInner(inner)}']`);
  next = next.replace(/\["((?:\\.|[^"\\])*)"\]/g, (_, inner) => `['${normalizeInner(inner)}']`);
  next = next.replace(/^"((?:\\.|[^"\\])*)"$/g, (_, inner) => `'${normalizeInner(inner)}'`);
  return next;
}

function getReplacementRange(left: string, insertText: string): { start: number; end: number } {
  const dollarBracketMatch = left.match(/\$\s*\[\s*(['"])([^'"]*)$/);
  if (dollarBracketMatch) {
    const partial = dollarBracketMatch[2] ?? "";
    return { start: left.length - (partial.length + 1), end: left.length };
  }

  const dollarTriggerMatch = left.match(/\$\s*\[\s*$|\$\s*$/);
  if (dollarTriggerMatch && dollarTriggerMatch.index !== undefined) {
    return { start: dollarTriggerMatch.index, end: left.length };
  }

  const dotMatch = left.match(/(.+?)\.\s*([$A-Za-z_][$A-Za-z0-9_]*)?$/);
  if (dotMatch) {
    const memberPrefix = dotMatch[2] ?? "";
    let start = left.length - memberPrefix.length;
    if (insertText.startsWith("[") && left[start - 1] === ".") {
      start -= 1;
    }
    return { start, end: left.length };
  }

  const identMatch = left.match(/[$A-Za-z_][$A-Za-z0-9_]*$/);
  if (identMatch) {
    return { start: left.length - identMatch[0].length, end: left.length };
  }

  return { start: left.length, end: left.length };
}

const shouldInsertAsSnippet = (text: string) => /\$\{\d+:/u.test(text) || /\$\d/u.test(text);

// ---------------------------------------------------------------------------
// Provider registration
// ---------------------------------------------------------------------------

export function registerYamlExpressionProvider(monaco: Monaco): IDisposable {
  return monaco.languages.registerCompletionItemProvider("yaml", {
    triggerCharacters: ["$", ".", "[", "'", '"', "{"],

    provideCompletionItems(model, position) {
      const exampleObj = exampleObjRef.current;
      if (!exampleObj) return { suggestions: [] };

      const fullText = model.getValue();
      const cursorOffset = model.getOffsetAt(position);
      const exprCtx = getExpressionContext(fullText, cursorOffset);
      if (!exprCtx) return { suggestions: [] };

      const wordAtPosition = model.getWordUntilPosition(position);
      const wordRange = new monaco.Range(
        position.lineNumber,
        wordAtPosition.startColumn,
        position.lineNumber,
        wordAtPosition.endColumn,
      );

      const rawSuggestions = getSuggestions(exprCtx.expressionText, exprCtx.expressionCursor, exampleObj, {
        allowInStrings: false,
        limit: 100,
      }).sort((a, b) => {
        const ap = SUGGESTION_SORT_PRIORITY[a.label];
        const bp = SUGGESTION_SORT_PRIORITY[b.label];
        if (ap !== undefined && bp !== undefined) return ap - bp;
        if (ap !== undefined) return -1;
        if (bp !== undefined) return 1;
        return a.label.localeCompare(b.label);
      });

      const left = exprCtx.expressionText.slice(0, exprCtx.expressionCursor);
      const leftFilterMatch = left.match(/[$A-Za-z_][$A-Za-z0-9_]*$|\$$/);
      const fallbackFilterText = leftFilterMatch?.[0] ?? wordAtPosition.word;

      const items: MonacoLanguages.CompletionItem[] = rawSuggestions.map((suggestion) => {
        const insertText = normalizeBracketQuotes(getInsertText(suggestion));
        const replacement = getReplacementRange(left, insertText);
        const startOffset = exprCtx.startOffset + replacement.start;
        const endOffset = exprCtx.startOffset + replacement.end;
        const startPos = model.getPositionAt(startOffset);
        const endPos = model.getPositionAt(endOffset);
        let range = new monaco.Range(startPos.lineNumber, startPos.column, endPos.lineNumber, endPos.column);
        if (!range.containsPosition(position)) {
          range = wordRange;
        }

        const filterText = fallbackFilterText || suggestion.label;
        const priority = SUGGESTION_SORT_PRIORITY[suggestion.label];
        const sortText = priority !== undefined ? String(priority).padStart(4, "0") : suggestion.label.toLowerCase();

        return {
          label: suggestion.label,
          kind: getCompletionKind(monaco, suggestion.kind),
          insertText,
          range,
          detail: suggestion.detail ?? suggestion.kind,
          filterText,
          sortText,
          command: { id: "editor.action.triggerSuggest", title: "Trigger Suggest" },
          insertTextRules: shouldInsertAsSnippet(insertText)
            ? monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet
            : undefined,
        };
      });

      return { suggestions: items };
    },
  });
}
