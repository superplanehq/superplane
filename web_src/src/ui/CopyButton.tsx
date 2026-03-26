import { useState } from "react";
import { Check, Copy } from "lucide-react";

interface CopyButtonProps {
  text: string;
  dark?: boolean;
}

export function CopyButton({ text, dark }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={`p-1 rounded transition-colors shrink-0 ${
        dark ? "hover:bg-gray-700" : "hover:bg-gray-200 dark:hover:bg-gray-700"
      }`}
      title="Copy to clipboard"
    >
      {copied ? (
        <Check size={13} className={dark ? "text-green-400" : "text-green-600 dark:text-green-400"} />
      ) : (
        <Copy size={13} className={dark ? "text-gray-400 hover:text-gray-200" : "text-gray-400"} />
      )}
    </button>
  );
}
