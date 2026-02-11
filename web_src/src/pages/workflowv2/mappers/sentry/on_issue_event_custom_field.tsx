import { Button } from "@/components/ui/button";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { CustomFieldRenderer, NodeInfo } from "../types";

type WebhookMetadata = {
  url?: string;
};

type OnIssueEventConfiguration = {
  events?: string[];
};

async function copyToClipboard(value: string) {
  try {
    await navigator.clipboard.writeText(value);
    showSuccessToast("Copied to clipboard");
  } catch (_err) {
    showErrorToast("Failed to copy text");
  }
}

export const onIssueEventCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as WebhookMetadata | undefined;
    const configuration = node.configuration as OnIssueEventConfiguration | undefined;
    const webhookUrl = metadata?.url;

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Webhook setup</span>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Configure a Sentry internal integration webhook to call SuperPlane.
            </p>

            <div className="mt-3">
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                SuperPlane webhook URL
              </label>
              <div className="mt-1 flex items-center gap-2">
                <input
                  type="text"
                  value={webhookUrl || "Save the canvas to generate a webhook URL"}
                  readOnly
                  className="w-full text-xs text-gray-800 dark:text-gray-100 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono"
                />
                <Button
                  type="button"
                  variant="outline"
                  disabled={!webhookUrl}
                  onClick={() => {
                    if (!webhookUrl) return;
                    void copyToClipboard(webhookUrl);
                  }}
                  className="shrink-0"
                >
                  Copy
                </Button>
              </div>
            </div>

            <div className="mt-3">
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Steps
              </label>
              <ol className="list-decimal ml-5 space-y-1 text-sm text-gray-700 dark:text-gray-300 mt-1">
                <li>In Sentry, create an Internal Integration.</li>
                <li>Paste the webhook URL above into the integration webhook URL field.</li>
                <li>
                  Subscribe to Issue events (created/resolved/unresolved{configuration?.events?.length ? "" : ", etc."}
                  ).
                </li>
                <li>Trigger an issue change in Sentry and check Runs.</li>
              </ol>
            </div>

            <div className="mt-3">
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Required token scopes
              </label>
              <div className="mt-1 text-xs text-gray-800 dark:text-gray-100 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono whitespace-pre-wrap">
                org:read, org:write, project:read, project:write, event:read, event:write
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};
