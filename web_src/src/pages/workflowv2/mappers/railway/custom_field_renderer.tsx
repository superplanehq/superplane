import { useState } from "react";
import { CustomFieldRenderer, NodeInfo } from "../types";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";

interface OnDeploymentEventMetadata {
  project?: {
    id: string;
    name: string;
  };
  webhookUrl?: string;
  webhookConfigUrl?: string;
}

/**
 * Copy button component for code blocks
 */
const CopyButton: React.FC<{ text: string }> = ({ text }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_err) {
      showErrorToast("Failed to copy text");
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1 bg-white outline-1 outline-black/20 hover:outline-black/30 rounded text-gray-600 dark:text-gray-400"
      title={copied ? "Copied!" : "Copy to clipboard"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
    </button>
  );
};

/**
 * Custom field renderer for Railway On Deployment Event trigger
 * Shows the webhook URL that needs to be configured in Railway
 */
export const onDeploymentEventCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnDeploymentEventMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    const curlExample = `curl -X POST \\
  -H "Content-Type: application/json" \\
  --data '{"type":"Deployment.deployed","details":{"status":"SUCCESS"},"resource":{"deployment":{"id":"test-123"}}}' \\
  ${webhookUrl}`;

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Railway Webhook Configuration</span>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Copy this URL and add it to your Railway project's webhook settings.
            </p>

            {/* Webhook URL Copy Field */}
            <div className="mt-3">
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Webhook URL
              </label>
              <div className="relative group mt-1">
                <input
                  type="text"
                  value={webhookUrl}
                  readOnly
                  className="w-full text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-purple-950/20 px-2.5 py-2 bg-purple-50 dark:bg-purple-800 rounded-md font-mono"
                />
                <CopyButton text={webhookUrl} />
              </div>
            </div>

            {/* Configure Webhook Button */}
            {metadata?.webhookConfigUrl && (
              <div className="mt-4">
                <a
                  href={metadata.webhookConfigUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white text-sm font-medium rounded-md transition-colors"
                >
                  <Icon name="external-link" size="sm" />
                  Configure Webhook in Railway
                </a>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                  Paste the webhook URL above and select "Deploy" events.
                </p>
              </div>
            )}

            {/* Test Command */}
            {metadata?.webhookUrl && (
              <div className="relative group mt-3">
                <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                  Test Command
                </label>
                <div className="relative group mt-1">
                  <pre className="text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-purple-950/20 px-2.5 py-2 bg-purple-50 dark:bg-purple-800 rounded-md font-mono whitespace-pre overflow-x-auto">
                    {curlExample}
                  </pre>
                  <CopyButton text={curlExample} />
                </div>
              </div>
            )}

            {!metadata?.webhookUrl && (
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-3">
                Save the canvas to generate the webhook URL.
              </p>
            )}
          </div>
        </div>
      </div>
    );
  },
};
