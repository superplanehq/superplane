import { useState } from "react";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/lib/toast";

export function CopyWebhookUrlButton({ webhookUrl }: { webhookUrl: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(webhookUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      showErrorToast("Failed to copy webhook URL");
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="inline-flex items-center gap-1.5 rounded border-1 border-edge-default bg-surface-raised px-2 py-1 text-xs font-medium text-content-secondary hover:bg-surface-subtle"
      title={copied ? "Copied!" : "Copy webhook URL"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
      {copied ? "Copied" : "Copy"}
    </button>
  );
}
