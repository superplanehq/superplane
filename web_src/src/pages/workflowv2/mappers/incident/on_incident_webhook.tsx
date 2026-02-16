import { useState } from "react";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";
import { CustomFieldRenderer, NodeInfo } from "../types";

interface OnIncidentMetadata {
  webhookUrl?: string;
}

function CopyWebhookUrlButton({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      showErrorToast("Failed to copy URL");
    }
  };
  return (
    <button
      type="button"
      onClick={handleCopy}
      className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1 bg-white dark:bg-gray-800 outline-1 outline-black/20 hover:outline-black/30 rounded text-gray-600 dark:text-gray-400"
      title={copied ? "Copied!" : "Copy to clipboard"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
    </button>
  );
}

export const onIncidentCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnIncidentMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";
    const isGenerated = Boolean(metadata?.webhookUrl);
    const isHttp = isGenerated && webhookUrl.startsWith("http://");

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">incident.io Webhook Setup</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              <ol className="list-decimal ml-4 space-y-1">
                <li>Save the canvas to generate the webhook URL below.</li>
                <li>
                  In incident.io go to <strong>Settings → Webhooks</strong> and create a new endpoint.
                </li>
                <li>Paste the webhook URL below into the endpoint URL field.</li>
                <li>
                  Subscribe to <strong>Public incident created (v2)</strong> and/or{" "}
                  <strong>Public incident updated (v2)</strong> to match the events selected above.
                </li>
                <li>
                  Copy the <strong>Signing secret</strong> from the new endpoint and paste it into the Signing secret
                  field above.
                </li>
              </ol>
              {isHttp && (
                <p className="mt-3 text-xs text-amber-700 dark:text-amber-300" role="status">
                  incident.io requires HTTPS. Set{" "}
                  <code className="bg-black/10 dark:bg-white/10 px-1 rounded">WEBHOOKS_BASE_URL</code> when starting the
                  app, then re-save to get an HTTPS URL.
                </p>
              )}
              {isGenerated && webhookUrl.includes("localhost") && (
                <p className="mt-3 text-xs text-amber-700 dark:text-amber-300" role="status">
                  This URL points to localhost, so incident.io cannot reach it. For local testing, use a tunnel (e.g.{" "}
                  <code className="bg-black/10 dark:bg-white/10 px-1 rounded">ngrok http 8000</code>
                  ), set{" "}
                  <code className="bg-black/10 dark:bg-white/10 px-1 rounded">WEBHOOKS_BASE_URL</code> to the tunnel
                  HTTPS URL, restart the app, then re-save the canvas.
                </p>
              )}
              <div className="mt-3">
                <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Webhook URL</span>
                <div className="relative group mt-1">
                  <input
                    type="text"
                    value={webhookUrl}
                    readOnly
                    className="w-full text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono"
                  />
                  {isGenerated && <CopyWebhookUrlButton code={webhookUrl} />}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};
