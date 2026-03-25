import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import Editor from "@monaco-editor/react";
import JsonView from "@uiw/react-json-view";
import { Copy, Check, Maximize2 } from "lucide-react";
import { useState, type ReactNode } from "react";
import type { BuildingBlock } from "./index";
import type { SuperplaneComponentsOutputChannel } from "@/api-client";

interface BuildingBlockPreviewProps {
  block: BuildingBlock;
  children: ReactNode;
}

export function BuildingBlockPreview({ block, children }: BuildingBlockPreviewProps) {
  const [isPayloadOpen, setIsPayloadOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const examplePayload = block.exampleOutput || block.exampleData;
  const hasPayload = examplePayload && Object.keys(examplePayload).length > 0;
  const hasDescription = !!block.description;
  const payloadLabel = block.type === "trigger" ? "Example Data" : "Example Output";

  const outputChannels = (block.outputChannels || []).filter(
    (ch): ch is SuperplaneComponentsOutputChannel => "label" in ch || "description" in ch,
  );

  const hasContent = hasDescription || outputChannels.length > 0 || hasPayload;
  if (!hasContent) return <>{children}</>;

  const payloadString = hasPayload ? JSON.stringify(examplePayload, null, 2) : "";
  const lineCount = payloadString.split("\n").length;
  const lineHeight = 19;
  const editorHeight = Math.min(Math.max(lineCount * lineHeight + 10, 100), 500);

  const handleCopy = () => {
    navigator.clipboard.writeText(payloadString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <>
      <HoverCard openDelay={400} closeDelay={150}>
        <HoverCardTrigger asChild>{children}</HoverCardTrigger>
        <HoverCardContent side="left" align="start" className="w-80 p-0 overflow-hidden max-h-[400px] overflow-y-auto">
          <div className="p-3 space-y-2.5">
            <div>
              <p className="text-sm font-medium text-gray-900">{block.label || block.name}</p>
              {block.description && <p className="text-xs text-gray-500 mt-1 leading-relaxed">{block.description}</p>}
            </div>

            {outputChannels.length > 0 && (
              <div>
                <p className="text-[11px] font-medium text-gray-400 uppercase tracking-wide mb-1">Output Channels</p>
                <div className="space-y-0.5">
                  {outputChannels.map((ch) => (
                    <div key={ch.name} className="flex items-baseline gap-1.5">
                      <span className="text-xs font-mono text-gray-700">{ch.label || ch.name}</span>
                      {ch.description && <span className="text-[11px] text-gray-400 truncate">{ch.description}</span>}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {hasPayload && (
              <div>
                <div className="flex items-center justify-between mb-1">
                  <p className="text-[11px] font-medium text-gray-400 uppercase tracking-wide">{payloadLabel}</p>
                  <button
                    className="p-1 text-gray-500 hover:text-gray-800"
                    onClick={(e) => {
                      e.stopPropagation();
                      e.preventDefault();
                      setIsPayloadOpen(true);
                    }}
                  >
                    <Maximize2 size={12} />
                  </button>
                </div>
                <div className="max-h-48 overflow-auto rounded">
                  <JsonView
                    value={examplePayload}
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
          </div>
        </HoverCardContent>
      </HoverCard>

      <Dialog open={isPayloadOpen} onOpenChange={setIsPayloadOpen}>
        <DialogContent
          size="large"
          className="w-[60vw] max-w-[60vw] h-auto max-h-[80vh] flex flex-col"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle>{block.label || block.name}</DialogTitle>
              <DialogDescription className="text-sm text-gray-500">{payloadLabel}</DialogDescription>
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
