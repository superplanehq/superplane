import React from "react";
import Editor from "@monaco-editor/react";
import type { editor, languages, IDisposable } from "monaco-editor";
import { FieldRendererProps } from "./types";
import { ConfigurationFieldRenderer } from "./index";
import { resolveIcon } from "@/lib/utils";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import {
  buildLookupPath,
  flattenForAutocomplete,
  getAutocompleteSuggestions,
  getAutocompleteSuggestionsWithTypes,
  isValidIdentifier,
  parsePathSegments,
} from "@/components/AutoCompleteInput/core";

export const ObjectFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  appInstallationId,
  organizationId,
  hasError,
  autocompleteExampleObj,
}) => {
  const [jsonError, setJsonError] = React.useState<string | null>(null);
  const [editorValue, setEditorValue] = React.useState<string>(() =>
    JSON.stringify(value === undefined ? {} : value, null, 2),
  );
  const [isModalOpen, setIsModalOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const autocompleteExampleRef = React.useRef<Record<string, unknown> | null>(autocompleteExampleObj ?? null);
  const flattenedDataRef = React.useRef<Record<string, string[]>>({});
  const previousEditorValueRef = React.useRef<string>(editorValue);
  const isApplyingAutoInsertRef = React.useRef(false);
  const completionProviderRef = React.useRef<IDisposable | null>(null);

  const objectOptions = field.typeOptions?.object;
  const schema = objectOptions?.schema;
  const hasSchema = !!schema && schema.length > 0;

  const copyToClipboard = () => {
    navigator.clipboard.writeText(editorValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

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

  const setupAutocomplete = React.useCallback(
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
        completionProviderRef.current = monaco.languages.registerCompletionItemProvider("json", {
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
              const nextSuggestionsAreArraySuggestions = nextSuggestions.some((suggestion) =>
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

      const contentDisposable = editorInstance.onDidChangeModelContent((event) => {
        if (isApplyingAutoInsertRef.current) {
          return;
        }

        const model = editorInstance.getModel();
        if (!model || event.changes.length !== 1) {
          return;
        }

        const change = event.changes[0];
        if (change.rangeLength !== 0) {
          return;
        }

        const currentValue = model.getValue();
        const previousValue = previousEditorValueRef.current;
        if (currentValue.length !== previousValue.length + 1) {
          return;
        }

        const position = editorInstance.getPosition();
        if (!position) {
          return;
        }

        const offset = model.getOffsetAt(position);
        const beforeCursor = currentValue.slice(0, offset);
        const afterCursor = currentValue.slice(offset);

        if (change.text === "{") {
          if (!beforeCursor.endsWith(startWord) || afterCursor.startsWith("}")) {
            return;
          }

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
          return;
        }

        if (isAllowedToSuggest(currentValue, offset)) {
          editorInstance.trigger("autocomplete", "editor.action.triggerSuggest", {});
        }
      });

      editorInstance.onDidDispose(() => {
        contentDisposable.dispose();
      });
    },
    [],
  );

  if (!hasSchema) {
    // Fallback to Monaco Editor if no schema defined
    const handleEditorChange = (newValue: string | undefined) => {
      const valueToUse = newValue || "{}";
      previousEditorValueRef.current = valueToUse;
      setEditorValue(valueToUse);
      try {
        const parsed = JSON.parse(valueToUse);
        onChange(parsed);
        setJsonError(null);
      } catch (error) {
        setJsonError("Invalid JSON format");
      }
    };

    const editorOptions = {
      minimap: { enabled: false },
      fontSize: 13,
      lineNumbers: "on" as const,
      wordWrap: "on" as const,
      folding: true,
      bracketPairColorization: {
        enabled: true,
      },
      autoIndent: "advanced" as const,
      formatOnPaste: true,
      formatOnType: true,
      tabSize: 2,
      insertSpaces: true,
      scrollBeyondLastLine: false,
      renderWhitespace: "boundary" as const,
      smoothScrolling: true,
      cursorBlinking: "smooth" as const,
      contextmenu: true,
      selectOnLineNumbers: true,
      suggestOnTriggerCharacters: true,
      quickSuggestions: {
        other: true,
        strings: true,
        comments: false,
      },
      wordBasedSuggestions: "off" as const,
    };

    return (
      <>
        <div className="flex flex-col gap-2 relative">
          <div
            className={`border rounded-md overflow-hidden ${hasError ? "border-red-500 border-2" : "border-gray-300 dark:border-gray-700"}`}
            style={{ height: "200px" }}
          >
            <div className="absolute right-1.5 top-1.5 z-10 flex items-center gap-1">
              <SimpleTooltip content={copied ? "Copied!" : "Copy"} hideOnClick={false}>
                <button onClick={copyToClipboard} className="p-1 rounded text-gray-500 hover:text-gray-800">
                  {React.createElement(resolveIcon("copy"), { size: 14 })}
                </button>
              </SimpleTooltip>
              <SimpleTooltip content="Expand">
                <button onClick={() => setIsModalOpen(true)} className="p-1 text-gray-500 hover:text-gray-800">
                  {React.createElement(resolveIcon("maximize-2"), { size: 14 })}
                </button>
              </SimpleTooltip>
            </div>
            <Editor
              height="100%"
              defaultLanguage="json"
              value={editorValue}
              onChange={handleEditorChange}
              onMount={setupAutocomplete}
              theme="vs"
              options={editorOptions}
            />
          </div>
          {jsonError && <p className="text-red-600 dark:text-red-400 text-xs">{jsonError}</p>}
        </div>

        {/* Expanded Editor Modal */}
        <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
          <DialogContent className="max-w-4xl max-h-[90vh] flex flex-col" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <DialogTitle>{field.label || field.name}</DialogTitle>
              <SimpleTooltip content={copied ? "Copied!" : "Copy"} hideOnClick={false}>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    copyToClipboard();
                  }}
                  className="px-3 py-1 text-sm text-gray-800 bg-gray-50 hover:bg-gray-200 rounded flex items-center gap-1"
                >
                  {React.createElement(resolveIcon("copy"), { size: 14 })}
                  Copy
                </button>
              </SimpleTooltip>
            </div>
            <div className="flex-1 overflow-auto border border-gray-200 dark:border-gray-700 rounded-md">
              <Editor
                height="600px"
                defaultLanguage="json"
                value={editorValue}
                onChange={handleEditorChange}
                onMount={setupAutocomplete}
                theme="vs"
                options={{
                  ...editorOptions,
                  automaticLayout: true,
                }}
              />
            </div>
          </DialogContent>
        </Dialog>
      </>
    );
  }

  const objValue = (value as Record<string, unknown>) ?? {};

  return (
    <div className="border border-gray-300 dark:border-gray-700 rounded-md p-4 space-y-4">
      {schema.map((schemaField) => (
        <ConfigurationFieldRenderer
          key={schemaField.name}
          field={schemaField}
          value={objValue[schemaField.name!]}
          onChange={(val) => {
            const newValue: Record<string, unknown> = { ...objValue, [schemaField.name!]: val };
            onChange(newValue);
          }}
          allValues={objValue}
          domainId={domainId}
          domainType={domainType}
          appInstallationId={appInstallationId}
          organizationId={organizationId}
          autocompleteExampleObj={autocompleteExampleObj}
        />
      ))}
    </div>
  );
};
