import type { ConfigurationField } from "@/api-client";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import Editor from "@monaco-editor/react";
import JsonView from "@uiw/react-json-view";
import { Copy, Check, Maximize2 } from "lucide-react";
import { useState } from "react";

interface DocsTabProps {
  description?: string;
  examplePayload?: Record<string, unknown>;
  payloadLabel?: string;
  configurationFields?: ConfigurationField[];
}

export function DocsTab({
  description,
  examplePayload,
  payloadLabel = "Example Output",
  configurationFields = [],
}: DocsTabProps) {
  const [isPayloadOpen, setIsPayloadOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const hasPayload = examplePayload && Object.keys(examplePayload).length > 0;
  const hasConfig = configurationFields.length > 0;

  const payloadString = hasPayload ? JSON.stringify(examplePayload, null, 2) : "";
  const lineCount = payloadString.split("\n").length;
  const lineHeight = 19;
  const editorHeight = Math.min(Math.max(lineCount * lineHeight + 10, 100), 500);

  const handleCopy = () => {
    navigator.clipboard.writeText(payloadString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (!description && !hasPayload && !hasConfig) {
    return (
      <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
        <p className="text-sm text-gray-500">No documentation available for this component.</p>
      </div>
    );
  }

  return (
    <>
      <div className="pb-8">
        {description && (
          <div className="w-full px-4 pt-3 pb-3">
            <span className="text-[13px] font-medium text-gray-500">Description</span>
            <p className="text-[13px] text-gray-800 mt-1 leading-relaxed">{description}</p>
          </div>
        )}

        {hasPayload && (
          <div className="w-full px-2 py-2 border-t border-gray-200">
            <div className="flex items-center justify-between mb-2 relative">
              <span className="text-[13px] font-medium text-gray-500 px-2">{payloadLabel}</span>
              <div className="flex items-center gap-1">
                <button onClick={handleCopy} className="p-1 text-gray-500 hover:text-gray-800">
                  {copied ? <Check size={16} /> : <Copy size={16} />}
                </button>
                <button onClick={() => setIsPayloadOpen(true)} className="p-1 text-gray-500 hover:text-gray-800">
                  <Maximize2 size={16} />
                </button>
              </div>
            </div>
            <div className="max-h-64 overflow-auto rounded">
              <JsonView
                value={examplePayload!}
                style={{
                  fontSize: "12px",
                  fontFamily:
                    'Monaco, Menlo, "Cascadia Code", "Segoe UI Mono", "Roboto Mono", Consolas, "Courier New", monospace',
                  backgroundColor: "#ffffff",
                  color: "#24292e",
                  padding: "8px",
                }}
                className="json-viewer-hide-types"
                displayObjectSize={false}
                enableClipboard={false}
              />
            </div>
          </div>
        )}

        {hasConfig && (
          <div className="w-full px-4 pt-3 pb-3 border-t border-gray-200">
            <span className="text-[13px] font-medium text-gray-500">Configuration</span>
            <div className="overflow-x-auto mt-2">
              <table className="text-xs w-full border-collapse">
                <thead>
                  <tr>
                    <th className="text-left py-1.5 px-2 border-b border-gray-200 font-medium text-gray-700 bg-gray-50">
                      Field
                    </th>
                    <th className="text-left py-1.5 px-2 border-b border-gray-200 font-medium text-gray-700 bg-gray-50">
                      Type
                    </th>
                    <th className="text-left py-1.5 px-2 border-b border-gray-200 font-medium text-gray-700 bg-gray-50">
                      Description
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {configurationFields.map((field) => (
                    <tr key={field.name}>
                      <td className="py-1.5 px-2 border-b border-gray-100 font-mono text-gray-700">
                        {field.label || field.name}
                        {field.required && <span className="text-red-400 ml-0.5">*</span>}
                      </td>
                      <td className="py-1.5 px-2 border-b border-gray-100 text-gray-500">{field.type || "string"}</td>
                      <td className="py-1.5 px-2 border-b border-gray-100 text-gray-500">{field.description || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      <Dialog open={isPayloadOpen} onOpenChange={setIsPayloadOpen}>
        <DialogContent
          size="large"
          className="w-[60vw] max-w-[60vw] h-auto max-h-[80vh] flex flex-col"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle>{payloadLabel}</DialogTitle>
              <DialogDescription className="sr-only">Example payload viewer.</DialogDescription>
            </div>
            <button
              onClick={handleCopy}
              className="px-3 py-1 text-sm text-gray-800 bg-gray-50 hover:bg-gray-200 rounded flex items-center gap-1"
            >
              {copied ? <Check size={14} /> : <Copy size={14} />}
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
          <div className="flex-1 overflow-hidden border border-gray-200 dark:border-gray-700 rounded-md">
            <Editor
              height={`${editorHeight}px`}
              defaultLanguage="json"
              value={payloadString}
              theme="vs"
              options={{
                readOnly: true,
                minimap: { enabled: false },
                fontSize: 13,
                lineNumbers: "on",
                wordWrap: "on",
                folding: true,
                scrollBeyondLastLine: false,
                renderWhitespace: "none",
                contextmenu: true,
                domReadOnly: true,
                scrollbar: {
                  vertical: "auto",
                  horizontal: "auto",
                },
              }}
            />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
