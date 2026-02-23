import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";
import { useState } from "react";

export function CopyButton({ text, label }: { text: string; label: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_error) {
      showErrorToast(`Failed to copy ${label}`);
    }
  };

  return (
    <button
      type="button"
      onClick={() => void handleCopy()}
      className="inline-flex items-center gap-1.5 px-2 py-1 text-xs font-medium text-blue-900 border border-blue-300 rounded bg-white hover:bg-blue-50"
      title={copied ? "Copied" : `Copy ${label}`}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
    </button>
  );
}

export function URLField({ label, url }: { label: string; url: string }) {
  return (
    <div>
      <div className="mb-1.5 text-xs font-medium text-blue-900/90">{label}</div>
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1 rounded border border-blue-200 bg-white px-2.5 py-2">
          <code
            className="block flex-1 overflow-hidden text-ellipsis whitespace-nowrap font-mono text-xs leading-5 text-blue-900"
            title={url}
          >
            {url}
          </code>
        </div>
        <CopyButton text={url} label={label} />
      </div>
    </div>
  );
}
