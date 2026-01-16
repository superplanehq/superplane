import { useCallback, useEffect, useRef } from "react";
import type { Monaco } from "@monaco-editor/react";
import type { editor as MonacoEditor, IDisposable, languages as MonacoLanguages } from "monaco-editor";
import { getSuggestions } from "@/components/AutoCompleteInput/core";

type ModelContext = {
  exampleObj: Record<string, unknown> | null;
  startWord: string;
  prefix: string;
  suffix: string;
};

type UseMonacoExpressionAutocompleteProps = {
  autocompleteExampleObj?: Record<string, unknown> | null;
  languageId: string;
  startWord?: string;
  prefix?: string;
  suffix?: string;
  allowOutsideExpression?: boolean;
};

const modelContextMap = new WeakMap<MonacoEditor.ITextModel, ModelContext>();
const providerRegistry = new Map<string, IDisposable>();
const triggerSuggestCommand = { id: "editor.action.triggerSuggest", title: "Trigger Suggest" };

const getCompletionKind = (
  monaco: Monaco,
  kind: ReturnType<typeof getSuggestions>[number]["kind"],
): MonacoLanguages.CompletionItemKind => {
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
};

const getSuggestionInsertText = (suggestion: ReturnType<typeof getSuggestions>[number]) => {
  if (suggestion.kind === "function") {
    return suggestion.insertText ?? `${suggestion.label}()`;
  }
  return suggestion.insertText ?? suggestion.label;
};

const shouldInsertAsSnippet = (insertText: string) => /\$\{\d+:/u.test(insertText) || /\$\d/u.test(insertText);

const normalizeBracketQuotes = (insertText: string) => {
  const normalizeInner = (value: string) => value.replace(/\\"/g, '"').replace(/'/g, "\\'");

  let next = insertText;
  next = next.replace(/\$\["((?:\\.|[^"\\])*)"\]/g, (_, inner) => `$['${normalizeInner(inner)}']`);
  next = next.replace(/\["((?:\\.|[^"\\])*)"\]/g, (_, inner) => `['${normalizeInner(inner)}']`);
  next = next.replace(/^"((?:\\.|[^"\\])*)"$/g, (_, inner) => `'${normalizeInner(inner)}'`);
  return next;
};

const shouldTriggerForKey = (event: any, monaco: Monaco) => {
  if (event.ctrlKey || event.metaKey || event.altKey) {
    return false;
  }

  const key = event.browserEvent.key;
  if (typeof key === "string") {
    if (key.length === 1) return true;
    if (key === "Backspace" || key === "Delete" || key === "Enter" || key === "Tab" || key === " ") return true;
    return false;
  }

  return (
    event.keyCode === monaco.KeyCode.Backspace ||
    event.keyCode === monaco.KeyCode.Delete ||
    event.keyCode === monaco.KeyCode.Enter ||
    event.keyCode === monaco.KeyCode.Tab ||
    event.keyCode === monaco.KeyCode.Space
  );
};

const getExpressionContext = (text: string, cursor: number, startWord: string, suffix: string) => {
  const suffixToken = suffix.trimStart();
  const openIndex = text.lastIndexOf(startWord, cursor);
  if (openIndex === -1) {
    return null;
  }

  const closeIndex = text.indexOf(suffixToken, openIndex + startWord.length);
  if (closeIndex !== -1 && cursor > closeIndex) {
    return null;
  }

  const startOffset = openIndex + startWord.length;
  const endOffset = closeIndex === -1 ? text.length : closeIndex;
  return {
    expressionText: text.slice(startOffset, endOffset),
    expressionCursor: Math.max(0, cursor - startOffset),
    startOffset,
    endOffset,
  };
};

const isAllowedToSuggest = (text: string, position: number, startWord: string, suffix: string) => {
  const suffixToken = suffix.trimStart();
  const openIndex = text.lastIndexOf(startWord, position);
  if (openIndex === -1) {
    return false;
  }

  const closeIndex = text.indexOf(suffixToken, openIndex + startWord.length);
  if (closeIndex !== -1 && position > closeIndex) {
    return false;
  }

  return true;
};

const getReplacementRange = (left: string, insertText: string) => {
  const envBracketMatch = left.match(/\$env\s*\[\s*(['"])([^'"]*)$/);
  if (envBracketMatch) {
    const partial = envBracketMatch[2] ?? "";
    return { start: left.length - (partial.length + 1), end: left.length };
  }

  const dollarBracketMatch = left.match(/\$\s*\[\s*(['"])([^'"]*)$/);
  if (dollarBracketMatch) {
    const partial = dollarBracketMatch[2] ?? "";
    return { start: left.length - (partial.length + 1), end: left.length };
  }

  const envTriggerMatch = left.match(/\$env\s*\[\s*$/);
  if (envTriggerMatch && envTriggerMatch.index !== undefined) {
    return { start: envTriggerMatch.index, end: left.length };
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
};

export const useMonacoExpressionAutocomplete = ({
  autocompleteExampleObj,
  languageId,
  startWord = "{{",
  prefix = "{{ $",
  suffix = " }}",
  allowOutsideExpression = false,
}: UseMonacoExpressionAutocompleteProps) => {
  const modelsRef = useRef<Set<MonacoEditor.ITextModel>>(new Set());
  const previousValueRef = useRef<WeakMap<MonacoEditor.ITextModel, string>>(new WeakMap());
  const applyingEditRef = useRef<Set<MonacoEditor.ITextModel>>(new Set());

  useEffect(() => {
    const nextExample = autocompleteExampleObj ?? null;
    for (const model of modelsRef.current) {
      const existing = modelContextMap.get(model);
      if (existing) {
        modelContextMap.set(model, { ...existing, exampleObj: nextExample, startWord, prefix, suffix });
      }
    }
  }, [autocompleteExampleObj, startWord, prefix, suffix]);

  const handleEditorMount = useCallback(
    (editor: MonacoEditor.IStandaloneCodeEditor, monaco: Monaco) => {
      const model = editor.getModel();
      if (!model) {
        return;
      }

      modelsRef.current.add(model);
      modelContextMap.set(model, { exampleObj: autocompleteExampleObj ?? null, startWord, prefix, suffix });
      previousValueRef.current.set(model, model.getValue());

      if (!providerRegistry.has(languageId)) {
        const disposable = monaco.languages.registerCompletionItemProvider(languageId, {
          triggerCharacters: ["$", ".", "[", "'", '"'],
          provideCompletionItems: (completionModel, position) => {
            const context = modelContextMap.get(completionModel);
            if (!context) {
              return { suggestions: [] };
            }

            const fullText = completionModel.getValue();
            const cursorOffset = completionModel.getOffsetAt(position);
            const expressionContext = allowOutsideExpression
              ? {
                  expressionText: fullText,
                  expressionCursor: cursorOffset,
                  startOffset: 0,
                  endOffset: fullText.length,
                }
              : getExpressionContext(fullText, cursorOffset, context.startWord, context.suffix);

            if (!expressionContext) {
              return { suggestions: [] };
            }

            const wordAtPosition = completionModel.getWordUntilPosition(position);
            const currentFilterText = wordAtPosition.word;
            const wordRange = new monaco.Range(
              position.lineNumber,
              wordAtPosition.startColumn,
              position.lineNumber,
              wordAtPosition.endColumn,
            );

            const suggestions = getSuggestions(
              expressionContext.expressionText,
              expressionContext.expressionCursor,
              context.exampleObj ?? {},
              { allowInStrings: allowOutsideExpression },
            );

            const left = expressionContext.expressionText.slice(0, expressionContext.expressionCursor);
            const leftFilterMatch = left.match(/[$A-Za-z_][$A-Za-z0-9_]*$|\$$/);
            const fallbackFilterText = leftFilterMatch?.[0] ?? currentFilterText;
            const items = suggestions.map((suggestion) => {
              const insertText = normalizeBracketQuotes(getSuggestionInsertText(suggestion));
              const replacement = getReplacementRange(left, insertText);
              const startOffset = expressionContext.startOffset + replacement.start;
              const endOffset = expressionContext.startOffset + replacement.end;
              const startPosition = completionModel.getPositionAt(startOffset);
              const endPosition = completionModel.getPositionAt(endOffset);
              let range = new monaco.Range(
                startPosition.lineNumber,
                startPosition.column,
                endPosition.lineNumber,
                endPosition.column,
              );
              if (!range.containsPosition(position)) {
                range = wordRange;
              }

              const filterText = fallbackFilterText || suggestion.label;

              return {
                label: suggestion.label,
                kind: getCompletionKind(monaco, suggestion.kind),
                insertText,
                range,
                detail: suggestion.detail ?? suggestion.kind,
                filterText,
                command: triggerSuggestCommand,
                insertTextRules: shouldInsertAsSnippet(insertText)
                  ? monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet
                  : undefined,
              };
            });

            return { suggestions: items };
          },
        });

        providerRegistry.set(languageId, disposable);
      }

      const triggerSuggestIfInsideExpression = (modelStartWord: string, modelSuffix: string) => {
        const position = editor.getPosition();
        if (!position) {
          return;
        }

        const cursorOffset = model.getOffsetAt(position);
        const expressionContext = allowOutsideExpression
          ? {
              expressionText: model.getValue(),
              expressionCursor: cursorOffset,
              startOffset: 0,
              endOffset: model.getValue().length,
            }
          : getExpressionContext(model.getValue(), cursorOffset, modelStartWord, modelSuffix);
        if (!expressionContext) {
          return;
        }

        requestAnimationFrame(() => {
          editor.trigger("expression-autocomplete", "editor.action.triggerSuggest", {});
        });
      };

      const changeDisposable = editor.onDidChangeModelContent((event) => {
        const modelContext = modelContextMap.get(model);
        if (!modelContext) {
          return;
        }

        if (applyingEditRef.current.has(model)) {
          applyingEditRef.current.delete(model);
          previousValueRef.current.set(model, model.getValue());
          return;
        }

        const { startWord: modelStartWord, suffix: modelSuffix, prefix: modelPrefix } = modelContext;
        const previousValue = previousValueRef.current.get(model) ?? "";
        const currentValue = model.getValue();
        let shouldTriggerSuggest = false;
        for (const change of event.changes) {
          if (change.text !== "{" || change.rangeLength !== 0 || change.rangeOffset < 1) {
            if (change.rangeLength === 0 && change.text) {
              for (const char of change.text) {
                if (char === "$" || char === "." || char === "[" || char === "'" || char === '"') {
                  shouldTriggerSuggest = true;
                  break;
                }
              }
            }
            if (shouldTriggerSuggest) {
              break;
            }
            continue;
          }

          const insertOffset = change.rangeOffset;
          const beforeChar = previousValue[insertOffset - 1];
          if (beforeChar !== "{") {
            continue;
          }

          if (isAllowedToSuggest(previousValue, insertOffset, modelStartWord, modelSuffix)) {
            continue;
          }

          if (currentValue.slice(insertOffset - 1, insertOffset + 1) !== "{{") {
            continue;
          }

          const replaceStart = insertOffset - 1;
          const replaceEnd = insertOffset + 1;
          const startPosition = model.getPositionAt(replaceStart);
          const endPosition = model.getPositionAt(replaceEnd);

          applyingEditRef.current.add(model);
          editor.executeEdits("expression-autocomplete", [
            {
              range: new monaco.Range(
                startPosition.lineNumber,
                startPosition.column,
                endPosition.lineNumber,
                endPosition.column,
              ),
              text: `${modelPrefix}${modelSuffix}`,
            },
          ]);

          const cursorOffset = replaceStart + modelPrefix.length;
          const cursorPosition = model.getPositionAt(cursorOffset);
          editor.setPosition(cursorPosition);
          editor.focus();
          editor.trigger("expression-autocomplete", "editor.action.triggerSuggest", {});
        }

        if (shouldTriggerSuggest) {
          triggerSuggestIfInsideExpression(modelStartWord, modelSuffix);
        }

        triggerSuggestIfInsideExpression(modelStartWord, modelSuffix);

        previousValueRef.current.set(model, model.getValue());
      });

      const keydownDisposable = editor.onKeyDown((event) => {
        const modelContext = modelContextMap.get(model);
        if (!modelContext) {
          return;
        }

        if (!shouldTriggerForKey(event, monaco)) {
          return;
        }

        triggerSuggestIfInsideExpression(modelContext.startWord, modelContext.suffix);
      });

      editor.onDidDispose(() => {
        changeDisposable.dispose();
        keydownDisposable.dispose();
        modelsRef.current.delete(model);
      });

      model.onWillDispose(() => {
        modelsRef.current.delete(model);
      });
    },
    [allowOutsideExpression, autocompleteExampleObj, languageId, prefix, startWord, suffix],
  );

  return { handleEditorMount };
};
