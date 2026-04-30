import { useEffect, useState } from "react";
import { Check, Copy } from "lucide-react";

interface CopyButtonProps {
  text: string;
  dark?: boolean;
}

export function CopyButton({ text, dark }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const id = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(id);
  }, [copied]);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();

    if (!text || !navigator.clipboard?.writeText) return;

    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={`p-1 rounded transition-colors shrink-0 ${
        dark ? "hover:bg-gray-700" : "hover:bg-gray-200 dark:hover:bg-gray-700"
      }`}
      aria-label={copied ? "Copied!" : "Copy to clipboard"}
      title={copied ? "Copied!" : "Copy to clipboard"}
    >
      <span role="status" className="sr-only">
        {copied ? "Copied to clipboard" : ""}
      </span>
      {copied ? (
        <Check size={13} className={dark ? "text-green-400" : "text-green-600 dark:text-green-400"} />
      ) : (
        <Copy size={13} className={dark ? "text-gray-400 hover:text-gray-200" : "text-gray-400"} />
      )}
    </button>
  );
}
