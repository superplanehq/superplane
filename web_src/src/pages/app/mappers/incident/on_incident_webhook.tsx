import type { CustomFieldRenderer, NodeInfo } from "../types";

import { CopyWebhookUrlButton } from "./copyWebhookUrlButton";
import { SetSigningSecretSection } from "./setSigningSecretSection";

interface OnIncidentConfig {
  events?: string[];
  signingSecretConfigured?: boolean;
}

interface OnIncidentMetadata {
  webhookUrl?: string;
  signingSecretConfigured?: boolean;
}

export const onIncidentCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const config = node.configuration as OnIncidentConfig | undefined;
    const metadata = node.metadata as OnIncidentMetadata | undefined;
    // Prefer config (persisted with canvas) so it works without workflow_nodes metadata merge
    const webhookConfigured = config?.signingSecretConfigured === true || metadata?.signingSecretConfigured === true;

    if (webhookConfigured) {
      return (
        <div className="border-t-1 border-gray-200 dark:border-gray-600 pt-4">
          <div className="text-xs text-gray-600 dark:text-gray-400 rounded-md border border-gray-200 dark:border-gray-600 px-3 py-2.5 bg-gray-50 dark:bg-gray-800/50">
            <p className="font-medium text-gray-700 dark:text-gray-300 mb-1">incident.io webhook is configured</p>
            <p>
              For security, the webhook URL and signing secret are not shown. To use a different URL or secret, add a
              new <strong>On Incident</strong> trigger and configure it there.
            </p>
          </div>
        </div>
      );
    }

    const webhookUrl = metadata?.webhookUrl || "URL will appear here after you save the canvas.";
    return (
      <div className="border-t-1 border-gray-200 dark:border-gray-600 pt-4">
        <div className="space-y-3">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">incident.io Webhook Setup</span>
          <div className="text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md space-y-2">
            <div className="flex items-center justify-between gap-2">
              <span className="font-medium text-gray-700 dark:text-gray-200">URL for incident.io</span>
              {metadata?.webhookUrl && <CopyWebhookUrlButton webhookUrl={metadata.webhookUrl} />}
            </div>
            <pre className="mt-1 text-xs font-mono whitespace-pre-wrap break-all text-gray-800 dark:text-gray-100">
              {webhookUrl}
            </pre>
            <p className="text-gray-600 dark:text-gray-400">
              In incident.io go to <strong>Settings → Webhooks</strong>, create an endpoint with this URL, and subscribe
              to <strong>Public incident created (v2)</strong> and <strong>Public incident updated (v2)</strong>. Then
              use <strong>Set signing secret</strong> below to store the signing secret from your incident.io webhook
              endpoint (it will not be stored in the workflow configuration).
            </p>
          </div>
          <SetSigningSecretSection nodeId={node.id} />
          <div className="rounded-md border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950/40 px-3 py-2.5">
            <p className="text-xs font-medium text-amber-800 dark:text-amber-200 mb-1">
              This trigger is not operational until the webhook is set up.
            </p>
            <p className="text-xs text-amber-700 dark:text-amber-300">
              Save the canvas to generate the webhook URL, add it in incident.io, then use{" "}
              <strong>Set signing secret</strong> above with the signing secret from the endpoint. Until then, no events
              will be received.
            </p>
          </div>
        </div>
      </div>
    );
  },
};
