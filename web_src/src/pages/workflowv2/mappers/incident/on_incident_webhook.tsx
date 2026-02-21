import { useState } from "react";
import { CustomFieldRenderer, NodeInfo } from "../types";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";

interface OnIncidentMetadata {
  webhookUrl?: string;
  signingSecretConfigured?: boolean;
}

const CopyWebhookUrlButton: React.FC<{ webhookUrl: string }> = ({ webhookUrl }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(webhookUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_err) {
      showErrorToast("Failed to copy webhook URL");
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="inline-flex items-center gap-1.5 px-2 py-1 text-xs font-medium text-gray-700 dark:text-gray-200 border-1 border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-800"
      title={copied ? "Copied!" : "Copy webhook URL"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
      {copied ? "Copied" : "Copy"}
    </button>
  );
};

export const onIncidentCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnIncidentMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl || "URL will appear here after you save the canvas.";
    const webhookConfigured = metadata?.signingSecretConfigured === true;

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">incident.io Webhook Setup</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md space-y-2">
              <div className="flex items-center justify-between gap-2">
                <span className="font-medium text-gray-700 dark:text-gray-200">URL for incident.io</span>
                {metadata?.webhookUrl && <CopyWebhookUrlButton webhookUrl={metadata.webhookUrl} />}
              </div>
              <pre className="mt-1 text-xs font-mono whitespace-pre-wrap break-all text-gray-800 dark:text-gray-100">
                {webhookUrl}
              </pre>
              <p className="text-gray-600 dark:text-gray-400">
                In incident.io go to <strong>Settings → Webhooks</strong>, create an endpoint with this URL, and
                subscribe to <strong>Public incident created (v2)</strong> and{" "}
                <strong>Public incident updated (v2)</strong>. Then paste the signing secret in the{" "}
                <strong>Webhook signing secret</strong> field above (in this trigger&apos;s configuration).
              </p>
            </div>
          </div>
          {!webhookConfigured && (
            <div className="rounded-md border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950/40 px-3 py-2.5">
              <p className="text-xs font-medium text-amber-800 dark:text-amber-200 mb-1">
                This trigger is not operational until the webhook is set up.
              </p>
              <p className="text-xs text-amber-700 dark:text-amber-300">
                Save the canvas to generate the webhook URL, add it in incident.io, then paste the signing secret from
                the endpoint into the <strong>Webhook signing secret</strong> field above and save. Until then, no
                events will be received.
              </p>
            </div>
          )}
          {webhookConfigured && (
            <p className="text-xs text-gray-500 dark:text-gray-400">
              Webhook is configured. To change the signing secret, edit the <strong>Webhook signing secret</strong>{" "}
              field above and save the canvas.
            </p>
          )}
        </div>
      </div>
    );
  },
};
