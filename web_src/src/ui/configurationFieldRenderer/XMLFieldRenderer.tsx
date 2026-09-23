import React from "react";
import Editor from "@monaco-editor/react";
import type { FieldRendererProps } from "./types";
import { resolveIcon } from "@/lib/utils";
import { coerceMonacoValue } from "@/lib/monaco";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { useTheme } from "@/contexts/useTheme";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import { useMonacoExpressionAutocomplete } from "./useMonacoExpressionAutocomplete";

const xmlEditorOptions = {
  minimap: { enabled: false },
  fontSize: 13,
  lineNumbers: "on" as const,
  wordWrap: "on" as const,
  folding: true,
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
  bracketPairColorization: {
    enabled: true,
  },
  suggestOnTriggerCharacters: true,
  quickSuggestions: {
    other: true,
    strings: true,
    comments: false,
  },
  wordBasedSuggestions: "off" as const,
};

function XMLExpandedEditorDialog({
  open,
  onOpenChange,
  title,
  editorValue,
  monacoTheme,
  copied,
  onCopy,
  onChange,
  onMount,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  editorValue: string;
  monacoTheme: string;
  copied: boolean;
  onCopy: () => void;
  onChange: (value: string | undefined) => void;
  onMount: ReturnType<typeof useMonacoExpressionAutocomplete>["handleEditorMount"];
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] flex flex-col" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between">
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className="sr-only">Expanded XML editor for {title}.</DialogDescription>
          <SimpleTooltip content={copied ? "Copied!" : "Copy"} hideOnClick={false}>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="gap-1"
              onClick={(e) => {
                e.stopPropagation();
                onCopy();
              }}
            >
              {React.createElement(resolveIcon("copy"), { size: 14 })}
              Copy
            </Button>
          </SimpleTooltip>
        </div>
        <div className="flex-1 border border-gray-200 dark:border-gray-600 rounded-md">
          <Editor
            height="600px"
            defaultLanguage="xml"
            value={editorValue}
            onChange={onChange}
            onMount={onMount}
            theme={monacoTheme}
            options={{
              ...xmlEditorOptions,
              automaticLayout: true,
            }}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}

export const XMLFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, autocompleteExampleObj }) => {
  const [isModalOpen, setIsModalOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const [validationError, setValidationError] = React.useState<string | null>(null);
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";
  const { handleEditorMount } = useMonacoExpressionAutocomplete({
    autocompleteExampleObj,
    languageId: "xml",
  });

  const editorValue = coerceMonacoValue(value);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(editorValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const validateXML = (xmlString: string): boolean => {
    if (!xmlString.trim()) {
      setValidationError(null);
      return true;
    }

    try {
      const parser = new DOMParser();
      const xmlDoc = parser.parseFromString(xmlString, "text/xml");
      const parseError = xmlDoc.querySelector("parsererror");

      if (parseError) {
        setValidationError("Invalid XML format");
        return false;
      }

      setValidationError(null);
      return true;
    } catch {
      setValidationError("Invalid XML format");
      return false;
    }
  };

  const handleEditorChange = (newValue: string | undefined) => {
    const valueToUse = newValue || "";
    validateXML(valueToUse);
    onChange(valueToUse || undefined);
  };

  const fieldTitle = field.label || field.name || "";

  return (
    <>
      <div className="flex flex-col gap-2 relative">
        <div className="border rounded-md border-gray-300 dark:border-gray-600 p-1" style={{ height: "200px" }}>
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
            defaultLanguage="xml"
            value={editorValue}
            onChange={handleEditorChange}
            onMount={handleEditorMount}
            theme={monacoTheme}
            options={xmlEditorOptions}
          />
        </div>
        {validationError && <p className="text-red-600 dark:text-red-400 text-xs">{validationError}</p>}
      </div>

      <XMLExpandedEditorDialog
        open={isModalOpen}
        onOpenChange={setIsModalOpen}
        title={fieldTitle}
        editorValue={editorValue}
        monacoTheme={monacoTheme}
        copied={copied}
        onCopy={copyToClipboard}
        onChange={handleEditorChange}
        onMount={handleEditorMount}
      />
    </>
  );
};
