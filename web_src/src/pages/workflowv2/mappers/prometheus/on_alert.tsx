import { CustomFieldRenderer, NodeInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import { getDetailsForAlert } from "./base";
import { OnAlertConfiguration, OnAlertMetadata, PrometheusAlertPayload } from "./types";

const statusLabels: Record<string, string> = {
  firing: "Firing",
  resolved: "Resolved",
};

export const onAlertTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as PrometheusAlertPayload;
    const title = buildEventTitle(eventData);
    const subtitle = buildEventSubtitle(eventData, context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as PrometheusAlertPayload;
    return getDetailsForAlert(eventData);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnAlertConfiguration | undefined;
    const metadataItems = [];
    const metadata = node.metadata as OnAlertMetadata | undefined;

    if (configuration?.statuses && configuration.statuses.length > 0) {
      const formattedStatuses = configuration.statuses
        .map((status) => statusLabels[status] || status)
        .filter((status, index, values) => values.indexOf(status) === index);

      metadataItems.push({
        icon: "funnel",
        label: `Statuses: ${formattedStatuses.join(", ")}`,
      });
    }

    if (configuration?.alertNames && configuration.alertNames.length > 0) {
      const alertNames = configuration.alertNames.filter((value) => value.trim().length > 0);
      if (alertNames.length > 0) {
        metadataItems.push({
          icon: "bell",
          label:
            alertNames.length > 3
              ? `Alert Names: ${alertNames.length} selected`
              : `Alert Names: ${alertNames.join(", ")}`,
        });
      }
    }

    if (metadata?.webhookAuthEnabled) {
      metadataItems.push({
        icon: "lock",
        label: "Webhook Auth: Bearer",
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: prometheusIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems.slice(0, 3),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as PrometheusAlertPayload;
      props.lastEventData = {
        title: buildEventTitle(eventData),
        subtitle: buildEventSubtitle(eventData, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

export const onAlertCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnAlertMetadata | undefined;
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";
    const webhookAuthEnabled = metadata?.webhookAuthEnabled || false;
    const alertmanagerSnippet = buildAlertmanagerSnippet(webhookUrl, webhookAuthEnabled);
    const authHint = buildAuthHint(webhookAuthEnabled);

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Alertmanager Webhook Setup</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md">
              <ol className="list-decimal ml-4 space-y-1">
                <li>Save the canvas to generate the webhook URL.</li>
                <li>Copy the receiver snippet below into your `alertmanager.yml`.</li>
                <li>Reload Alertmanager config (for example, POST /-/reload when lifecycle reload is enabled).</li>
              </ol>
              <p className="mt-3">
                Receiver provisioning in upstream Alertmanager is config-based, so SuperPlane does not create receivers
                by API.
              </p>
              <p className="mt-2">{authHint}</p>
              <div className="mt-3">
                <span className="text-xs font-medium text-gray-700 dark:text-gray-200">Webhook URL</span>
                <pre className="mt-1 text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono whitespace-pre-wrap break-all">
                  {webhookUrl}
                </pre>
              </div>
              <div className="mt-3">
                <span className="text-xs font-medium text-gray-700 dark:text-gray-200">alertmanager.yml Snippet</span>
                <pre className="mt-1 text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono whitespace-pre-wrap break-all">
                  {alertmanagerSnippet}
                </pre>
              </div>
              <div className="mt-3">
                <div>
                  <span className="text-xs font-medium text-gray-700 dark:text-gray-200">
                    Reload Alertmanager config
                  </span>
                  <pre className="mt-1 text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-white dark:bg-gray-900 rounded-md font-mono whitespace-pre-wrap break-all">
                    curl -X POST https://alertmanager.example.com/-/reload
                  </pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};

function buildEventTitle(eventData: PrometheusAlertPayload): string {
  const alertName = eventData?.labels?.alertname || "Alert";
  const sourceParts = [eventData?.labels?.instance, eventData?.labels?.job].filter(Boolean);

  if (sourceParts.length > 0) {
    return `Alert ${eventData?.status} · ${alertName} · ${sourceParts.join(" · ")}`;
  }

  return `Alert ${eventData?.status} · ${alertName}`;
}

function buildEventSubtitle(eventData: PrometheusAlertPayload, createdAt?: string): string {
  const parts: string[] = [];

  const severity = eventData?.labels?.severity;
  if (severity) {
    parts.push(severity);
  }

  if (createdAt) {
    parts.push(formatTimeAgo(new Date(createdAt)));
  }

  return parts.join(" · ");
}

function buildAuthHint(webhookAuthEnabled: boolean): string {
  if (webhookAuthEnabled) {
    return "Use the same value from SuperPlane integration field Webhook Secret in Alertmanager http_config.authorization.credentials.";
  }

  return "Webhook bearer auth is disabled, so no auth block is needed in Alertmanager.";
}

function buildAlertmanagerSnippet(webhookUrl: string, webhookAuthEnabled: boolean): string {
  if (webhookAuthEnabled) {
    return `receivers:
  - name: superplane
    webhook_configs:
      - url: ${webhookUrl}
        send_resolved: true
        http_config:
          authorization:
            type: Bearer
            credentials: <webhook-secret>

route:
  receiver: superplane
  # ... other config ...`;
  }

  return `receivers:
  - name: superplane
    webhook_configs:
      - url: ${webhookUrl}
        send_resolved: true

route:
  receiver: superplane
  # ... other config ...`;
}
