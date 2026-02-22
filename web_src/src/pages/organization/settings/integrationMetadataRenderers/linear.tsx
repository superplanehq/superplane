import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";
import { useState } from "react";
import { IntegrationMetadataRenderer } from "./types";

function CopyButton({ text, label }: { text: string; label: string }) {
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

function URLField({ label, url }: { label: string; url: string }) {
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

export const linearMetadataRenderer: IntegrationMetadataRenderer = ({ integration }) => {
  const metadata = integration.status?.metadata as Record<string, unknown> | undefined;
  const webhookUrl = metadata?.webhookUrl;
  const callbackUrl = metadata?.callbackUrl;

  const hasWebhook = typeof webhookUrl === "string" && webhookUrl.trim().length > 0;
  const hasCallback = typeof callbackUrl === "string" && callbackUrl.trim().length > 0;

  if (!hasWebhook && !hasCallback) {
    return null;
  }

  return (
    <div className="rounded-md border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
      <div className="mb-3 font-medium">Linear OAuth Application URLs</div>
      <div className="mb-3 text-xs text-blue-900/90">
        Use these URLs when creating your OAuth2 application in{" "}
        <a
          href="https://linear.app/settings/api/applications/new"
          target="_blank"
          rel="noopener noreferrer"
          className="font-semibold [text-decoration:underline!important] [text-underline-offset:2px] [text-decoration-thickness:2px]"
        >
          Linear API Settings
        </a>
        .
      </div>
      <div className="space-y-3">
        {hasCallback && <URLField label="Callback URL" url={(callbackUrl as string).trim()} />}
        {hasWebhook && <URLField label="Webhook URL" url={(webhookUrl as string).trim()} />}
      </div>
    </div>
  );
};
