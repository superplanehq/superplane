import React, { useCallback, useState } from "react";
import { resolveIcon } from "@/lib/utils";
import { SimpleTooltip } from "./SimpleTooltip";

interface CopyPayloadButtonProps {
  payload: any;
  variant?: "icon" | "labeled";
  iconSize?: number;
  className?: string;
}

export const CopyPayloadButton: React.FC<CopyPayloadButtonProps> = ({
  payload,
  variant = "icon",
  iconSize = 14,
  className,
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    const text = typeof payload === "string" ? payload : JSON.stringify(payload, null, 2);
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [payload]);

  const defaultClassName =
    variant === "labeled"
      ? "px-3 py-1 text-sm text-gray-800 bg-gray-50 hover:bg-gray-200 rounded flex items-center gap-1"
      : "p-1 rounded text-gray-500 hover:text-gray-800";

  return (
    <SimpleTooltip content={copied ? "Copied!" : "Copy"} hideOnClick={false}>
      <button onClick={handleCopy} className={className ?? defaultClassName}>
        {React.createElement(resolveIcon("copy"), { size: iconSize })}
        {variant === "labeled" && "Copy"}
      </button>
    </SimpleTooltip>
  );
};
