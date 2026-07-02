import React from "react";
import { getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import type {
  TriggerRenderer,
  CustomFieldRenderer,
  NodeInfo,
  TriggerRendererContext,
  TriggerEventContext,
} from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { WebhookCustomFieldContent } from "./WebhookCustomFieldContent";

const DEFAULT_HEADER_TOKEN_NAME = "X-Webhook-Token";
const DEFAULT_SIGNATURE_HEADER = "X-Signature-256";

interface WebhookConfiguration {
  authentication?: string;
  headerName?: string;
  signatureHeader?: string;
}

interface WebhookMetadata {
  url?: string;
  authentication?: string;
}

function formatAuthenticationMethod(auth: string, options?: { headerName?: string; signatureHeader?: string }): string {
  switch (auth) {
    case "none":
      return "No authentication";
    case "signature":
      return `HMAC signature (${options?.signatureHeader || DEFAULT_SIGNATURE_HEADER})`;
    case "bearer":
      return "Bearer token";
    case "header_token":
      return `Header token (${options?.headerName || DEFAULT_HEADER_TOKEN_NAME})`;
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

function getWebhookDisplayHeaderValues(headers: Record<string, string>): Record<string, string> {
  const values: Record<string, string> = {};
  if (headers["User-Agent"]) values["User-Agent"] = headers["User-Agent"];
  if (headers["Content-Type"]) values["Content-Type"] = headers["Content-Type"];
  return values;
}

function appendWebhookRequestValues(values: Record<string, string>, webhookData: WebhookEventData): void {
  const method = webhookData.method || webhookData.headers?.Method;
  if (method) values.Method = method;
  if (webhookData.url) values.URL = webhookData.url;
  if (webhookData.headers) {
    Object.assign(values, getWebhookDisplayHeaderValues(webhookData.headers));
  }
}

function getWebhookEventTitle(context: TriggerEventContext): string {
  // Check for run_name in the webhook request body
  const runName = (context.event?.data?.data as { body?: { run_name?: string } })?.body?.run_name;
  if (runName) {
    return `${runName}`;
  }

  // Fallback to method and date
  return `Webhook ${context.event?.id} at ${new Date(context.event?.createdAt || "").toLocaleString()}`;
}

/**
 * Renderer for the "webhook" trigger type
 */
export const webhookTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    return {
      title: getWebhookEventTitle(context),
      subtitle: renderTimeAgo(new Date(context.event?.createdAt || "")),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const webhookData = context.event?.data?._webhook as WebhookEventData | undefined;
    const receivedOn = (context.event?.data?.["timestamp"] as string) || context.event?.createdAt;
    const values: Record<string, string> = {
      "Received on": receivedOn ? new Date(receivedOn).toLocaleString() : "n/a",
      Response: "200",
    };

    if (webhookData) {
      appendWebhookRequestValues(values, webhookData);
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as WebhookMetadata | undefined;
    const configuration = node.configuration as WebhookConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSlug: definition.icon || "webhook",
      iconColor: getColorClass("black"),
      collapsedBackground: "bg-white",
      metadata: [
        {
          icon: "link",
          label: formatWebhookUrl(metadata),
        },
        {
          icon: "shield-check",
          label: formatAuthenticationMethod(metadata?.authentication || configuration?.authentication || "none", {
            headerName: configuration?.headerName,
            signatureHeader: configuration?.signatureHeader,
          }),
        },
      ],
    };

    if (lastEvent) {
      const eventDate = new Date(lastEvent.createdAt);

      props.lastEventData = {
        title: getWebhookEventTitle({ event: lastEvent }),
        subtitle: renderTimeAgo(eventDate),
        receivedAt: eventDate,
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

export const webhookCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as WebhookMetadata | undefined;
    const config = node.configuration as WebhookConfiguration | undefined;
    return <WebhookCustomFieldContent nodeId={node.id || ""} metadata={metadata} config={config} />;
  },
};
