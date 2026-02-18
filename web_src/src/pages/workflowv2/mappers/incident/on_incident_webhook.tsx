import { CustomFieldRenderer, NodeInfo, type CustomFieldRendererContext } from "../types";
import { Icon } from "@/components/Icon";

const integrationsPath = `/${typeof window !== "undefined" ? window.location.pathname.split("/")[1] || "" : ""}/settings/integrations`;

export const onIncidentCustomFieldRenderer: CustomFieldRenderer = {
  render: (_node: NodeInfo, context?: CustomFieldRendererContext) => {
    const integration = context?.integration;
    const webhookConfigured =
      integration?.status?.metadata &&
      typeof integration.status.metadata.webhookSigningSecretConfigured === "boolean" &&
      integration.status.metadata.webhookSigningSecretConfigured === true;

    return (
      <div className="border-t border-gray-200 dark:border-gray-700 pt-4 mt-4 space-y-2">
        <p className="text-xs font-medium text-gray-700 dark:text-gray-300">Webhook</p>
        {!webhookConfigured && (
          <div className="rounded-md border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950/40 px-3 py-2.5">
            <p className="text-xs font-medium text-amber-800 dark:text-amber-200 mb-1">
              This trigger is not operational until the webhook is set up.
            </p>
            <p className="text-xs text-amber-700 dark:text-amber-300">
              Complete the setup in <strong>Settings → Integrations</strong>: open your incident integration, copy the
              webhook URL, add it in incident.io, and paste the signing secret there. Until then, no events will be
              received.
            </p>
            <a
              href={integrationsPath}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 mt-2 text-xs font-medium text-amber-800 dark:text-amber-200 hover:underline"
            >
              Open Settings → Integrations
              <Icon name="external-link" size="sm" />
            </a>
          </div>
        )}
        {webhookConfigured && (
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Webhook is configured in your integration. To change it, go to <strong>Settings → Integrations</strong>.
          </p>
        )}
      </div>
    );
  },
};
