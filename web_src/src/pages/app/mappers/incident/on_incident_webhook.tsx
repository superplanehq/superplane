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
        <div className="border-t-1 border-edge-default pt-4">
          <div className="rounded-md border border-edge-default bg-surface-subtle px-3 py-2.5 text-xs text-content-secondary">
            <p className="mb-1 font-medium text-content-primary">incident.io webhook is configured</p>
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
      <div className="border-t-1 border-edge-default pt-4">
        <div className="space-y-3">
          <span className="text-sm font-medium text-content-secondary">incident.io Webhook Setup</span>
          <div className="space-y-2 rounded-md border-1 border-edge-default bg-surface-subtle px-2.5 py-2 text-xs text-content-primary">
            <div className="flex items-center justify-between gap-2">
              <span className="font-medium text-content-secondary">URL for incident.io</span>
              {metadata?.webhookUrl && <CopyWebhookUrlButton webhookUrl={metadata.webhookUrl} />}
            </div>
            <pre className="mt-1 font-mono text-xs text-content-primary whitespace-pre-wrap break-all">
              {webhookUrl}
            </pre>
            <p className="text-content-secondary">
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
