import { useState } from "react";
import { CustomFieldRenderer, NodeInfo } from "../types";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";

export interface OnVMCreatedMetadata {
  webhookUrl?: string;
}

const CopyCodeButton: React.FC<{ code: string }> = ({ code }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
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

export const onVMCreatedCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnVMCreatedMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Webhook URL for trigger</span>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Use this URL as the HTTP destination in Eventarc or as the push subscription URL for a Pub/Sub topic.
            </p>
            <div className="mt-3">
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Webhook URL
              </label>
              <div className="relative group mt-1">
                <input
                  type="text"
                  value={webhookUrl}
                  readOnly
                  className="w-full text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md font-mono"
                />
                <CopyCodeButton code={webhookUrl} />
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};
