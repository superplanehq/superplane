import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";

interface WebhookConfiguration {
  url?: string;
  authentication?: string;
  signatureKey?: string;
  bearerToken?: string;
  apiKeyName?: string;
  apiKeyValue?: string;
}

interface WebhookMetadata {
  url?: string;
  authentication?: string;
  signatureKey?: string;
  bearerToken?: string;
  apiKeyName?: string;
  apiKeyValue?: string;
}

function formatAuthenticationMethod(auth: string): string {
  switch (auth) {
    case "none":
      return "No authentication";
    case "signature":
      return "HMAC signature";
    case "bearer":
      return "Bearer token";
    case "apikey":
      return "API key";
    default:
      return "Unknown authentication";
  }
}

function formatWebhookUrl(metadata?: WebhookMetadata): string {
  if (!metadata?.url) {
    return "URL will be generated after setup";
  }

  // Truncate long URLs for display
  if (metadata.url.length > 50) {
    const parts = metadata.url.split("/");
    if (parts.length > 3) {
      return `${parts[0]}//${parts[2]}/.../${parts[parts.length - 1]}`;
    }
  }

  return metadata.url;
}

interface WebhookEventData {
  method?: string;
  url?: string;
  headers?: Record<string, string>;
}

function getWebhookEventTitle(event: WorkflowsWorkflowEvent): string {
  const webhookData = event.data?._webhook as WebhookEventData | undefined;
  const method = webhookData?.method || webhookData?.headers?.Method || "POST";
  const eventDate = new Date(event.createdAt!);

  const formattedDate = eventDate.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
    timeZoneName: "short",
  });

  return `Webhook: ${method} ${formattedDate}`;
}

/**
 * Renderer for the "webhook" trigger type
 */
export const webhookTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventDate = new Date(event.createdAt!);

    return {
      title: getWebhookEventTitle(event),
      subtitle: formatTimeAgo(eventDate),
    };
  },

  getRootEventValues: (event: WorkflowsWorkflowEvent): Record<string, string> => {
    const webhookData = event.data?._webhook as WebhookEventData | undefined;
    const values: Record<string, string> = {
      Timestamp: (event.data?.["timestamp"] as string) || event.createdAt || "n/a",
    };

    if (webhookData) {
      const method = webhookData.method || webhookData.headers?.Method;
      if (method) {
        values.Method = method;
      }
      if (webhookData.url) {
        values.URL = webhookData.url;
      }
      if (webhookData.headers) {
        // Add some common headers that might be useful
        const headers = webhookData.headers;
        if (headers["User-Agent"]) {
          values["User-Agent"] = headers["User-Agent"];
        }
        if (headers["Content-Type"]) {
          values["Content-Type"] = headers["Content-Type"];
        }
      }
    }

    return values;
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent?: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as WebhookMetadata | undefined;
    const configuration = node.configuration as WebhookConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name!,
      iconSlug: trigger.icon || "webhook",
      iconColor: getColorClass("black"),
      headerColor: "bg-white",
      collapsedBackground: "bg-white",
      metadata: [
        {
          icon: "link",
          label: formatWebhookUrl(metadata),
        },
        {
          icon: "shield-check",
          label: formatAuthenticationMethod(metadata?.authentication || configuration?.authentication || "none"),
        },
      ],
    };

    if (lastEvent) {
      const eventDate = new Date(lastEvent.createdAt!);

      props.lastEventData = {
        title: getWebhookEventTitle(lastEvent),
        subtitle: formatTimeAgo(eventDate),
        receivedAt: eventDate,
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
