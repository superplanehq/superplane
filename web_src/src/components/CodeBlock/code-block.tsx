import { useState } from "react";
import { Icon } from "@/components/Icon";

interface CodeBlockProps {
  children: string;
  className?: string;
}

export function CodeBlock({ children, className = "" }: CodeBlockProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(children);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy text: ", err);
    }
  };

  return (
    <pre
      className={`relative group bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-x-auto mb-2 cursor-pointer ${className}`}
      onClick={handleCopy}
      title={copied ? "Copied!" : "Click to copy"}
    >
      {children}
      <button
        onClick={(e) => {
          e.stopPropagation();
          handleCopy();
        }}
        className="absolute! top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1 bg-gray-200 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 rounded text-gray-600 dark:text-gray-400"
        title={copied ? "Copied!" : "Copy to clipboard"}
      >
        <Icon name={copied ? "check" : "content_copy"} size="sm" />
      </button>
    </pre>
  );
}
