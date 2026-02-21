import React from "react";
import Editor from "@monaco-editor/react";
import { FieldRendererProps } from "./types";
import { ConfigurationFieldRenderer } from "./index";
import { resolveIcon } from "@/lib/utils";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import { useMonacoExpressionAutocomplete } from "./useMonacoExpressionAutocomplete";
import { parseDefaultValues } from "../../utils/components";

export const ObjectFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  appInstallationId,
  organizationId,
  autocompleteExampleObj,
  allowExpressions = false,
}) => {
  const [jsonError, setJsonError] = React.useState<string | null>(null);
  const hasInitialized = React.useRef(false);
  const serializeValueForEditor = React.useCallback((input: unknown) => {
    if (typeof input === "string") {
      return input;
    }
    if (input === undefined || input === null) {
      return "";
    }
    return JSON.stringify(input, null, 2);
  }, []);
  const coerceDefaultValue = React.useCallback((input: unknown) => {
    if (typeof input !== "string") {
      return input;
    }
    try {
      return JSON.parse(input);
    } catch {
      return input;
    }
  }, []);
  const [editorValue, setEditorValue] = React.useState<string>(() => serializeValueForEditor(value));
  const [isModalOpen, setIsModalOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const { handleEditorMount } = useMonacoExpressionAutocomplete({
    autocompleteExampleObj,
    languageId: "json",
  });

  const objectOptions = field.typeOptions?.object;
  const schema = objectOptions?.schema;
  const hasSchema = !!schema && schema.length > 0;

  React.useEffect(() => {
    if (value !== undefined && value !== null) {
      hasInitialized.current = true;
    }

    if (!hasInitialized.current && (value === undefined || value === null) && field.defaultValue !== undefined) {
      hasInitialized.current = true;
      const defaultValue = coerceDefaultValue(field.defaultValue);
      onChange(defaultValue);
      setEditorValue(serializeValueForEditor(defaultValue));
      setJsonError(null);
    }
  }, [value, field.defaultValue, onChange, coerceDefaultValue, serializeValueForEditor]);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(editorValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (!hasSchema) {
    // Fallback to Monaco Editor if no schema defined
    const handleEditorChange = (newValue: string | undefined) => {
      const valueToUse = newValue ?? "";
      setEditorValue(valueToUse);
      const trimmedValue = valueToUse.trim();
      if (trimmedValue.length === 0) {
        onChange(undefined);
        setJsonError(null);
        return;
      }

      const normalizeExpressionsForValidation = (input: string) => input.replace(/{{[\s\S]*?}}/g, "{}");
      const hasExpressions = /{{[\s\S]*?}}/.test(valueToUse);

      try {
        const parsed = JSON.parse(valueToUse);
        onChange(parsed);
        setJsonError(null);
      } catch (error) {
        if (allowExpressions && hasExpressions) {
          try {
            JSON.parse(normalizeExpressionsForValidation(valueToUse));
            onChange(valueToUse);
            setJsonError(null);
            return;
          } catch {
            // fallthrough to error below
          }
        }
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
      quickSuggestions: false,
      wordBasedSuggestions: "off" as const,
      suggest: {
        showWords: false,
        showSnippets: false,
        showKeywords: false,
        showProperties: false,
        showValues: false,
        showText: false,
        showVariables: true,
        showFunctions: true,
        showFields: true,
        showMethods: false,
        showClasses: false,
        showModules: false,
        showInterfaces: false,
        showStructs: false,
        showEnums: false,
        showEnumMembers: false,
        showConstants: false,
        showEvents: false,
        showOperators: false,
        showFiles: false,
        showReferences: false,
        showFolders: false,
        showTypeParameters: false,
        showIssues: false,
        showUsers: false,
        showColors: false,
        showUnits: false,
      },
    };

    return (
      <>
        <div className="flex flex-col gap-2 relative">
          <div className="border rounded-md border-gray-300 dark:border-gray-700 p-2" style={{ height: "200px" }}>
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
              <DialogDescription className="sr-only">
                Expanded JSON editor for {field.label || field.name}.
              </DialogDescription>
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
            <div className="flex-1 overflow-auto rounded-md p-2 relative border border-gray-300 dark:border-gray-700">
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
                  renderValidationDecorations: "off",
                }}
              />
            </div>
          </DialogContent>
        </Dialog>
      </>
    );
  }

  // Merge schema defaults so visibility/required for nested fields see e.g. authMethod
  // Use parseDefaultValues to properly convert string defaults to their correct types
  // (e.g. boolean "false" -> false, number "5" -> 5)
  const schemaDefaults = React.useMemo(() => {
    if (!schema) return {};
    return parseDefaultValues(schema);
  }, [schema]);

  const objValue = React.useMemo(
    () => ({ ...schemaDefaults, ...((value as Record<string, unknown>) ?? {}) }),
    [schemaDefaults, value],
  );

  // When value is missing or empty object, push schema defaults to parent so required-object
  // validation (e.g. "schedule is required") sees a non-empty value and doesn't flag the field.
  const hasPushedSchemaDefaults = React.useRef(false);
  React.useEffect(() => {
    if (hasPushedSchemaDefaults.current) return;
    const isEmpty =
      value === undefined ||
      value === null ||
      (typeof value === "object" && value !== null && Object.keys(value).length === 0);
    if (!isEmpty || Object.keys(schemaDefaults).length === 0) return;
    hasPushedSchemaDefaults.current = true;
    onChange(schemaDefaults);
  }, [value, schemaDefaults, onChange]);

  return (
    <div className="border border-gray-300 dark:border-gray-700 rounded-md p-4 space-y-4">
      {schema.map((schemaField) => (
        <ConfigurationFieldRenderer
          allowExpressions={allowExpressions}
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
