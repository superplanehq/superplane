import React from "react";
import Editor from "@monaco-editor/react";
import { FieldRendererProps } from "./types";
import { resolveIcon } from "@/lib/utils";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";

export const TextFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const [isModalOpen, setIsModalOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  const copyToClipboard = () => {
    const textToCopy = (value as string) || "";
    navigator.clipboard.writeText(textToCopy);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const editorValue = (value as string) || "";

  const handleEditorChange = (newValue: string | undefined) => {
    const valueToUse = newValue || "";
    onChange(valueToUse || undefined);
  };

  const editorOptions = {
    minimap: { enabled: false },
    fontSize: 13,
    lineNumbers: "on" as const,
    wordWrap: "on" as const,
    folding: false,
    autoIndent: "none" as const,
    formatOnPaste: false,
    formatOnType: false,
    tabSize: 2,
    insertSpaces: true,
    scrollBeyondLastLine: false,
    renderWhitespace: "boundary" as const,
    smoothScrolling: true,
    cursorBlinking: "smooth" as const,
    contextmenu: true,
    selectOnLineNumbers: true,
    bracketPairColorization: {
      enabled: false,
    },
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
            defaultLanguage="plaintext"
            value={editorValue}
            onChange={handleEditorChange}
            theme="vs"
            options={editorOptions}
          />
        </div>
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
              defaultLanguage="plaintext"
              value={editorValue}
              onChange={handleEditorChange}
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
};
