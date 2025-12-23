import React from "react";
import Editor from "@monaco-editor/react";
import { Input } from "../input";
import { FieldRendererProps } from "./types";
import { resolveIcon } from "@/lib/utils";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const [isModalOpen, setIsModalOpen] = React.useState(false);

  // Detect if this field should use Monaco Editor based on field name
  const isMultilineField = field.name === "payloadText" || field.name === "payloadXML";
  const language = field.name === "payloadXML" ? "xml" : "plaintext";

  // Use Monaco Editor for multiline payload fields
  if (isMultilineField) {
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
      folding: language === "xml",
      autoIndent: (language === "xml" ? "advanced" : "none") as "advanced" | "none",
      formatOnPaste: language === "xml",
      formatOnType: language === "xml",
      tabSize: 2,
      insertSpaces: true,
      scrollBeyondLastLine: false,
      renderWhitespace: "boundary" as const,
      smoothScrolling: true,
      cursorBlinking: "smooth" as const,
      contextmenu: true,
      selectOnLineNumbers: true,
      bracketPairColorization: {
        enabled: language === "xml",
      },
    };

    return (
      <>
        <div className="flex flex-col gap-2 relative">
          <div
            className={`border rounded-md overflow-hidden ${hasError ? "border-red-500 border-2" : "border-gray-300 dark:border-gray-700"}`}
            style={{ height: "200px" }}
          >
            <div className="absolute right-2 top-2 z-10">
              <button
                onClick={() => setIsModalOpen(true)}
                className="p-1 text-gray-500 hover:text-gray-800 bg-white/80 hover:bg-white rounded border border-gray-300"
                title="Expand editor"
              >
                {React.createElement(resolveIcon("maximize-2"), { size: 14 })}
              </button>
            </div>
            <Editor
              height="100%"
              defaultLanguage={language}
              value={editorValue}
              onChange={handleEditorChange}
              theme="vs"
              options={editorOptions}
            />
          </div>
        </div>

        {/* Expanded Editor Modal */}
        <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
          <DialogContent className="max-w-6xl max-h-[90vh] flex flex-col">
            <DialogTitle>{field.label || field.name}</DialogTitle>
            <div className="flex-1 overflow-hidden border border-gray-200 dark:border-gray-700 rounded-md">
              <Editor
                height="600px"
                defaultLanguage={language}
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
  }

  // Use regular input for single-line fields
  return (
    <Input
      type={field.sensitive ? "password" : "text"}
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.placeholder || ""}
      className={hasError ? "border-red-500 border-2" : ""}
    />
  );
};
