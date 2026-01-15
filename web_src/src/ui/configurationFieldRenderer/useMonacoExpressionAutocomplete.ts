import React from "react";
import type { editor, IDisposable, languages } from "monaco-editor";
import {
  buildLookupPath,
  flattenForAutocomplete,
  getAutocompleteSuggestions,
  getAutocompleteSuggestionsWithTypes,
  isValidIdentifier,
  parsePathSegments,
} from "@/components/AutoCompleteInput/core";

interface UseMonacoExpressionAutocompleteArgs {
  autocompleteExampleObj?: Record<string, unknown> | null;
  languageId: string;
}

export const useMonacoExpressionAutocomplete = ({
  autocompleteExampleObj,
  languageId,
}: UseMonacoExpressionAutocompleteArgs) => {
  const autocompleteExampleRef = React.useRef<Record<string, unknown> | null>(autocompleteExampleObj ?? null);
  const flattenedDataRef = React.useRef<Record<string, string[]>>({});
  const previousValueRef = React.useRef<string>("");
  const isApplyingAutoInsertRef = React.useRef(false);
  const completionProviderRef = React.useRef<IDisposable | null>(null);

  React.useEffect(() => {
    autocompleteExampleRef.current = autocompleteExampleObj ?? null;
    flattenedDataRef.current = autocompleteExampleObj ? flattenForAutocomplete(autocompleteExampleObj) : {};
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

      const getWordAtCursor = (text: string, position: number) => {
        const beforeCursor = text.slice(0, position);
        const start = beforeCursor.lastIndexOf("$");
        if (start === -1) {
          return {
            word: "",
            start: position,
            end: position,
          };
        }

        let end = position;
        while (end < text.length && !isDelimiter(text[end])) {
          end += 1;
        }
        return {
          word: text.substring(start, end),
          start,
          end,
        };
      };

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

      const getNormalizedPath = (rawWord: string) => {
        return buildLookupPath(parsePathSegments(rawWord));
      };

      const getPathContext = (rawWord: string, normalizedWord: string) => {
        if (!normalizedWord) {
          return { basePath: "", lastKey: "" };
        }

        if (rawWord.endsWith(".")) {
          return { basePath: normalizedWord, lastKey: "" };
        }

        const parts = normalizedWord.split(".");
        return {
          basePath: parts.slice(0, -1).join("."),
          lastKey: parts[parts.length - 1] ?? "",
        };
      };

      const formatSuggestionLabel = (suggestion: string) => {
        if (suggestion.match(/\[/)) {
          return suggestion;
        }
        if (isValidIdentifier(suggestion)) {
          return suggestion;
        }
        return `['${suggestion}']`;
      };

      const formatDisplayPathWithSingleQuotes = (segments: Array<string | number>, includeDollar = false) => {
        let path = includeDollar ? "$" : "";
        segments.forEach((segment) => {
          if (typeof segment === "number") {
            path += `[${segment}]`;
            return;
          }

          if (isValidIdentifier(segment)) {
            path += path ? `.${segment}` : segment;
            return;
          }

          path += `['${segment}']`;
        });

        return path;
      };

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
            const { word, start, end } = getWordAtCursor(text, offset);
            if (!word) {
              return { suggestions: [] };
            }

            if (word.startsWith("[") && !word.startsWith("$")) {
              return { suggestions: [] };
            }

            if (!isAllowedToSuggest(text, offset)) {
              return { suggestions: [] };
            }

            const flattenedData = flattenedDataRef.current;
            const normalizedWord = getNormalizedPath(word);
            const { basePath, lastKey } = getPathContext(word, normalizedWord);
            const parsedInput = basePath;
            const filterTextBase = word || "";

            const newSuggestions = getAutocompleteSuggestionsWithTypes(
              flattenedData,
              parsedInput || "root",
              basePath,
              exampleObj,
            );
            const arraySuggestions = getAutocompleteSuggestionsWithTypes(
              flattenedData,
              parsedInput ? `${parsedInput}.${lastKey}` : lastKey,
              basePath,
              exampleObj,
            ).filter(({ suggestion }) => suggestion.match(/\[\d+\]$/));
            const similarSuggestions = newSuggestions.filter(
              ({ suggestion }) => suggestion.startsWith(lastKey) && suggestion !== lastKey,
            );

            const allSuggestionsMap = new Map<string, { suggestion: string; type: string }>();
            [...arraySuggestions, ...similarSuggestions].forEach((item) => {
              allSuggestionsMap.set(item.suggestion, item);
            });
            const allSuggestions = Array.from(allSuggestionsMap.values());

            const startPos = model.getPositionAt(start);
            const endPos = model.getPositionAt(end);
            const range = new monaco.Range(startPos.lineNumber, startPos.column, endPos.lineNumber, endPos.column);

            const suggestions: languages.CompletionItem[] = allSuggestions.map((suggestionItem) => {
              const normalizedPath = suggestionItem.suggestion.startsWith(basePath)
                ? suggestionItem.suggestion
                : basePath
                  ? `${basePath}.${suggestionItem.suggestion}`
                  : suggestionItem.suggestion;
              const nextSuggestions = getAutocompleteSuggestions(flattenedData, normalizedPath);
              const nextSuggestionsAreArraySuggestions = nextSuggestions.some((suggestion: string) =>
                suggestion.match(/\[\d+\]$/),
              );
              const isObjectKey = nextSuggestions.length > 0 && !nextSuggestionsAreArraySuggestions;
              const displayPath = formatDisplayPathWithSingleQuotes(parsePathSegments(normalizedPath), true);
              const insertText = isObjectKey ? `${displayPath}.` : displayPath;

              return {
                label: formatSuggestionLabel(suggestionItem.suggestion),
                kind: monaco.languages.CompletionItemKind.Field,
                detail: suggestionItem.type,
                insertText,
                filterText: filterTextBase,
                range,
                command: { id: "editor.action.triggerSuggest", title: "Trigger Suggest" },
              };
            });

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
