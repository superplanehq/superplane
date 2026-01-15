import React from "react";
import type { editor, IDisposable, languages } from "monaco-editor";
import { getSuggestions } from "@/components/AutoCompleteInput/core";

interface UseMonacoExpressionAutocompleteArgs {
  autocompleteExampleObj?: Record<string, unknown> | null;
  languageId: string;
}

export const useMonacoExpressionAutocomplete = ({
  autocompleteExampleObj,
  languageId,
}: UseMonacoExpressionAutocompleteArgs) => {
  const autocompleteExampleRef = React.useRef<Record<string, unknown> | null>(autocompleteExampleObj ?? null);
  const previousValueRef = React.useRef<string>("");
  const isApplyingAutoInsertRef = React.useRef(false);
  const completionProviderRef = React.useRef<IDisposable | null>(null);

  React.useEffect(() => {
    autocompleteExampleRef.current = autocompleteExampleObj ?? null;
  }, [autocompleteExampleObj]);

  React.useEffect(() => {
    return () => {
      completionProviderRef.current?.dispose();
      completionProviderRef.current = null;
    };
  }, []);

  const handleEditorMount = React.useCallback(
    (editorInstance: editor.IStandaloneCodeEditor, monaco: typeof import("monaco-editor")) => {
      const startWord = "{{";
      const prefix = "{{ $";
      const suffix = " }}";

      const isDelimiter = (char: string) =>
        /\s/.test(char) || char === "," || char === ":" || char === "{" || char === "}";

      const isAllowedToSuggest = (text: string, position: number) => {
        const openIndex = text.lastIndexOf(startWord, position);
        if (openIndex === -1) {
          return false;
        }

        const closeIndex = text.indexOf(suffix, openIndex + startWord.length);
        if (closeIndex !== -1 && position > closeIndex) {
          return false;
        }

        return true;
      };

      const getExpressionContext = (text: string, cursor: number) => {
        const openIndex = text.lastIndexOf(startWord, cursor);
        if (openIndex === -1) {
          return null;
        }

        const closeIndex = text.indexOf(suffix, openIndex + startWord.length);
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

      const getSuggestionInsertText = (suggestion: ReturnType<typeof getSuggestions>[number]) => {
        if (suggestion.kind === "function") {
          return `${suggestion.label}()`;
        }
        return suggestion.insertText ?? suggestion.label;
      };

      const getFilterText = (text: string, cursor: number) => {
        const beforeCursor = text.slice(0, cursor);
        const start = beforeCursor.lastIndexOf("$");
        if (start === -1) {
          return "";
        }

        let end = cursor;
        while (end < text.length && !isDelimiter(text[end])) {
          end += 1;
        }
        return text.substring(start, end);
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

      const shouldInsertAsSnippet = (text: string) => text.includes("${");

      if (!completionProviderRef.current) {
        completionProviderRef.current = monaco.languages.registerCompletionItemProvider(languageId, {
          triggerCharacters: [".", "[", "$"],
          provideCompletionItems: (model, position) => {
            const exampleObj = autocompleteExampleRef.current;
            if (!exampleObj) {
              return { suggestions: [] };
            }

            const text = model.getValue();
            const offset = model.getOffsetAt(position);
            const context = getExpressionContext(text, offset);
            if (!context || !isAllowedToSuggest(text, offset)) {
              return { suggestions: [] };
            }

            const newSuggestions = getSuggestions(context.expressionText, context.expressionCursor, exampleObj);
            const left = context.expressionText.slice(0, context.expressionCursor);
            const filterTextBase = getFilterText(context.expressionText, context.expressionCursor);

            const suggestions: languages.CompletionItem[] = newSuggestions.map((suggestionItem, index) => {
              const insertText = getSuggestionInsertText(suggestionItem);
              const replaceRange = getReplacementRange(left, insertText);
              const startPos = model.getPositionAt(context.startOffset + replaceRange.start);
              const endPos = model.getPositionAt(context.startOffset + replaceRange.end);
              const range = new monaco.Range(startPos.lineNumber, startPos.column, endPos.lineNumber, endPos.column);

              return {
                label: suggestionItem.label,
                kind: monaco.languages.CompletionItemKind.Field,
                detail: suggestionItem.detail ?? suggestionItem.kind,
                insertText,
                range,
                sortText: String(index).padStart(4, "0"),
                insertTextRules: shouldInsertAsSnippet(insertText)
                  ? monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet
                  : monaco.languages.CompletionItemInsertTextRule.None,
                command: { id: "editor.action.triggerSuggest", title: "Trigger Suggest" },
              };
            });

            console.log(suggestions);

            return { suggestions };
          },
        });
      }

      previousValueRef.current = editorInstance.getValue();
      const contentDisposable = editorInstance.onDidChangeModelContent((event) => {
        if (isApplyingAutoInsertRef.current) {
          return;
        }

        const model = editorInstance.getModel();
        if (!model) {
          return;
        }

        const position = editorInstance.getPosition();
        if (!position) {
          return;
        }

        const currentValue = model.getValue();
        const offset = model.getOffsetAt(position);
        const beforeCursor = currentValue.slice(0, offset);
        const afterCursor = currentValue.slice(offset);
        const previousValue = previousValueRef.current;

        if (event.changes.length === 1) {
          const change = event.changes[0];
          if (
            change.text === "{" &&
            change.rangeLength === 0 &&
            currentValue.length === previousValue.length + 1 &&
            beforeCursor.endsWith(startWord) &&
            !afterCursor.startsWith("}")
          ) {
            const replaceStart = offset - startWord.length;
            const startPos = model.getPositionAt(replaceStart);
            const endPos = model.getPositionAt(offset);

            isApplyingAutoInsertRef.current = true;
            model.pushEditOperations(
              [],
              [
                {
                  range: new monaco.Range(startPos.lineNumber, startPos.column, endPos.lineNumber, endPos.column),
                  text: `${prefix}${suffix}`,
                },
              ],
              () => null,
            );
            const cursorOffset = replaceStart + prefix.length;
            const cursorPosition = model.getPositionAt(cursorOffset);
            editorInstance.setPosition(cursorPosition);
            editorInstance.focus();
            editorInstance.trigger("autocomplete", "editor.action.triggerSuggest", {});
            isApplyingAutoInsertRef.current = false;
            previousValueRef.current = model.getValue();
            return;
          }
        }

        if (isAllowedToSuggest(currentValue, offset)) {
          editorInstance.trigger("autocomplete", "editor.action.triggerSuggest", {});
        }

        previousValueRef.current = currentValue;
      });

      editorInstance.onDidDispose(() => {
        contentDisposable.dispose();
      });
    },
    [languageId],
  );

  return { handleEditorMount };
};
