import type { SuperplaneComponentsOutputChannel } from "@/api-client";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";
import JsonView from "@uiw/react-json-view";
import { Maximize2 } from "lucide-react";
import { useState, type ReactNode } from "react";
import type { BuildingBlock } from "./index";
import { PayloadDialog } from "./PayloadDialog";

const jsonViewStyle = {
  fontSize: "12px",
  fontFamily: 'Monaco, Menlo, "Cascadia Code", "Segoe UI Mono", "Roboto Mono", Consolas, "Courier New", monospace',
  backgroundColor: "#ffffff",
  color: "#24292e",
  padding: "8px",
} as const;

interface BuildingBlockPreviewProps {
  block: BuildingBlock;
  children: ReactNode;
}

export function BuildingBlockPreview({ block, children }: BuildingBlockPreviewProps) {
  const [isPayloadOpen, setIsPayloadOpen] = useState(false);
  const examplePayload = block.exampleOutput || block.exampleData;
  const hasPayload = examplePayload && Object.keys(examplePayload).length > 0;
  const payloadLabel = block.type === "trigger" ? "Example Data" : "Example Output";

  const outputChannels = (block.outputChannels || []).filter(
    (ch): ch is SuperplaneComponentsOutputChannel => "label" in ch || "description" in ch,
  );

  if (!block.description && outputChannels.length === 0 && !hasPayload) {
    return <>{children}</>;
  }

  const payloadString = hasPayload ? JSON.stringify(examplePayload, null, 2) : "";

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
                    style={jsonViewStyle}
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

      <PayloadDialog
        open={isPayloadOpen}
        onOpenChange={setIsPayloadOpen}
        title={block.label || block.name}
        label={payloadLabel}
        payloadString={payloadString}
      />
    </>
  );
}
