import JsonView from "@uiw/react-json-view";
import { Copy, Check, Maximize2 } from "lucide-react";
import { useState } from "react";
import { useTheme } from "@/contexts/useTheme";
import { getJsonViewStyle, jsonViewClassName } from "@/lib/jsonViewTheme";
import { PayloadDialog } from "./PayloadDialog";

interface PayloadPreviewProps {
  value: Record<string, unknown>;
  label: string;
  dialogTitle: string;
  maxHeight?: string;
  showCopy?: boolean;
  labelSize?: "sm" | "md";
  /** When provided, the expand button calls this instead of opening an internal dialog. */
  onExpand?: () => void;
}

export function PayloadPreview({
  value,
  label,
  dialogTitle,
  maxHeight = "max-h-48",
  showCopy = false,
  labelSize = "sm",
  onExpand,
}: PayloadPreviewProps) {
  const { resolvedTheme } = useTheme();
  const [isExpanded, setIsExpanded] = useState(false);
  const [copied, setCopied] = useState(false);
  const payloadString = JSON.stringify(value, null, 2);
  const managesOwnDialog = !onExpand;
  const jsonViewStyle = getJsonViewStyle(resolvedTheme);

  const handleCopy = () => {
    navigator.clipboard.writeText(payloadString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const iconSize = labelSize === "md" ? 16 : 12;
  const labelClass =
    labelSize === "md"
      ? "text-[13px] font-medium text-gray-500 dark:text-gray-400"
      : "text-[11px] font-medium text-gray-400 uppercase tracking-wide dark:text-gray-500";

  return (
    <>
      <div className="flex items-center justify-between mb-1">
        <p className={labelClass}>{label}</p>
        <div className="flex items-center gap-1">
          {showCopy && (
            <button
              onClick={handleCopy}
              className="p-1 text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
            >
              {copied ? <Check size={iconSize} /> : <Copy size={iconSize} />}
            </button>
          )}
          <button
            className="p-1 text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              if (onExpand) {
                onExpand();
              } else {
                setIsExpanded(true);
              }
            }}
          >
            <Maximize2 size={iconSize} />
          </button>
        </div>
      </div>
      <div className={`${maxHeight} overflow-auto rounded`}>
        <JsonView
          value={value}
          style={{ ...jsonViewStyle, padding: "8px" }}
          className={jsonViewClassName}
          displayObjectSize={false}
          enableClipboard={false}
        />
      </div>

      {managesOwnDialog && (
        <PayloadDialog
          open={isExpanded}
          onOpenChange={setIsExpanded}
          title={dialogTitle}
          payloadString={payloadString}
        />
      )}
    </>
  );
}
