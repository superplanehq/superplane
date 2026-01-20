import React from "react";
import Editor from "@monaco-editor/react";
import { FieldRendererProps } from "./types";
import { ConfigurationFieldRenderer } from "./index";
import { resolveIcon } from "@/lib/utils";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import { useMonacoExpressionAutocomplete } from "./useMonacoExpressionAutocomplete";

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
  const { handleEditorMount } = useMonacoExpressionAutocomplete({
    autocompleteExampleObj,
    languageId: "json",
  });

  const objectOptions = field.typeOptions?.object;
  const schema = objectOptions?.schema;
  const hasSchema = !!schema && schema.length > 0;

  const copyToClipboard = () => {
    navigator.clipboard.writeText(editorValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (!hasSchema) {
    // Fallback to Monaco Editor if no schema defined
    const handleEditorChange = (newValue: string | undefined) => {
      const valueToUse = newValue || "{}";
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
            className="border rounded-md overflow-hidden border-gray-300 dark:border-gray-700"
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
              onMount={handleEditorMount}
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
                onMount={handleEditorMount}
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
