import React from "react";
import Editor from "@monaco-editor/react";
import { FieldRendererProps } from "./types";
import { ConfigurationFieldRenderer } from "./index";

export const ObjectFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  hasError,
}) => {
  const [isDarkMode, setIsDarkMode] = React.useState(false);
  const [jsonError, setJsonError] = React.useState<string | null>(null);
  const objectOptions = field.typeOptions?.object;
  const schema = objectOptions?.schema;
  const hasSchema = !!schema && schema.length > 0;

  // Detect dark mode
  React.useEffect(() => {
    const checkDarkMode = () => {
      setIsDarkMode(window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches);
    };

    checkDarkMode();

    const observer = new MutationObserver(checkDarkMode);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class"],
    });

    return () => observer.disconnect();
  }, []);

  if (!hasSchema) {
    // Fallback to Monaco Editor if no schema defined
    const handleEditorChange = (value: string | undefined) => {
      const newValue = value || "{}";
      try {
        const parsed = JSON.parse(newValue);
        onChange(parsed);
        setJsonError(null);
      } catch (error) {
        setJsonError("Invalid JSON format");
      }
    };

    return (
      <div className="flex flex-col gap-2">
        <div
          className={`border rounded-md overflow-hidden ${hasError ? "border-red-500 border-2" : "border-gray-300 dark:border-zinc-700"}`}
          style={{ height: "200px" }}
        >
          <Editor
            height="100%"
            defaultLanguage="json"
            value={JSON.stringify(value === undefined ? {} : value, null, 2)}
            onChange={handleEditorChange}
            theme={isDarkMode ? "vs-dark" : "vs"}
            options={{
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: "on",
              wordWrap: "on",
              folding: true,
              bracketPairColorization: {
                enabled: true,
              },
              autoIndent: "advanced",
              formatOnPaste: true,
              formatOnType: true,
              tabSize: 2,
              insertSpaces: true,
              scrollBeyondLastLine: false,
              renderWhitespace: "boundary",
              smoothScrolling: true,
              cursorBlinking: "smooth",
              contextmenu: true,
              selectOnLineNumbers: true,
            }}
          />
        </div>
        {jsonError && <p className="text-red-600 dark:text-red-400 text-xs">{jsonError}</p>}
      </div>
    );
  }

  const objValue = (value as Record<string, unknown>) ?? {};

  return (
    <div className="border border-gray-300 dark:border-zinc-700 rounded-md p-4 space-y-4">
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
        />
      ))}
    </div>
  );
};
