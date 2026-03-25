import JsonView from "@uiw/react-json-view";
import { Copy, Check, Maximize2 } from "lucide-react";
import { useState } from "react";
import { PayloadDialog } from "./PayloadDialog";

const jsonViewStyle = {
  fontSize: "12px",
  fontFamily: 'Monaco, Menlo, "Cascadia Code", "Segoe UI Mono", "Roboto Mono", Consolas, "Courier New", monospace',
  backgroundColor: "#ffffff",
  color: "#24292e",
  padding: "8px",
} as const;

interface PayloadPreviewProps {
  value: Record<string, unknown>;
  label: string;
  dialogTitle: string;
  maxHeight?: string;
  showCopy?: boolean;
  labelSize?: "sm" | "md";
}

export function PayloadPreview({
  value,
  label,
  dialogTitle,
  maxHeight = "max-h-48",
  showCopy = false,
  labelSize = "sm",
}: PayloadPreviewProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [copied, setCopied] = useState(false);
  const payloadString = JSON.stringify(value, null, 2);

  const handleCopy = () => {
    navigator.clipboard.writeText(payloadString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const labelClass =
    labelSize === "md"
      ? "text-[13px] font-medium text-gray-500"
      : "text-[11px] font-medium text-gray-400 uppercase tracking-wide";

  return (
    <>
      <div className="flex items-center justify-between mb-1">
        <p className={labelClass}>{label}</p>
        <div className="flex items-center gap-1">
          {showCopy && (
            <button onClick={handleCopy} className="p-1 text-gray-500 hover:text-gray-800">
              {copied ? <Check size={labelSize === "md" ? 16 : 12} /> : <Copy size={labelSize === "md" ? 16 : 12} />}
            </button>
          )}
          <button
            className="p-1 text-gray-500 hover:text-gray-800"
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              setIsExpanded(true);
            }}
          >
            <Maximize2 size={labelSize === "md" ? 16 : 12} />
          </button>
        </div>
      </div>
      <div className={`${maxHeight} overflow-auto rounded`}>
        <JsonView
          value={value}
          style={jsonViewStyle}
          className="json-viewer-hide-types"
          displayObjectSize={false}
          enableClipboard={false}
        />
      </div>

      <PayloadDialog
        open={isExpanded}
        onOpenChange={setIsExpanded}
        title={dialogTitle}
        label={label}
        payloadString={payloadString}
      />
    </>
  );
}
