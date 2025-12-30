import { useState } from "react";
import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer, CustomFieldRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";
import { Icon } from "@/components/Icon";

interface WebhookConfiguration {
  url?: string;
  authentication?: string;
  signatureKey?: string;
  headerKeyName?: string;
  headerKeyValue?: string;
}

interface WebhookMetadata {
  url?: string;
  authentication?: string;
  signatureKey?: string;
  headerKeyName?: string;
  headerKeyValue?: string;
}

function formatAuthenticationMethod(auth: string): string {
  switch (auth) {
    case "none":
      return "No authentication";
    case "signature":
      return "HMAC signature";
    case "headerkey":
      return "Header key";
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

/**
 * Copy button component for code blocks
 */
const CopyCodeButton: React.FC<{ code: string }> = ({ code }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy text: ", err);
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1 bg-gray-200 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 rounded text-gray-600 dark:text-gray-400"
      title={copied ? "Copied!" : "Copy to clipboard"}
    >
      <Icon name={copied ? "check" : "content_copy"} size="sm" />
    </button>
  );
};

/**
 * Custom field renderer for webhook component configuration
 */
export const webhookCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: ComponentsNode, configuration: Record<string, unknown>) => {
    const metadata = node.metadata as WebhookMetadata | undefined;
    const config = configuration as WebhookConfiguration | undefined;
    const authMethod = metadata?.authentication || config?.authentication || "none";
    const webhookUrl =
      metadata?.url ||
      "https://app.superplane.com/3ee1aa47-3a60-4c1f-b645-0b9859ab91f8/workflows/8d4dbb35-0034-499f-81da-226da05452e2/webhook/somestring-or-something";

    let description: string;
    let code: string;
    let title: string;

    switch (authMethod) {
      case "signature":
        title = "HMAC Signature Authentication";
        description = "Use HMAC SHA-256 signature to authenticate your webhook requests.";
        code = `export SIGNATURE_KEY="${metadata?.signatureKey || config?.signatureKey || "<your-signature-key>"}"
export PAYLOAD='{"hello":"world"}'

export SIGNATURE=$(echo -n "$PAYLOAD" \\
  | openssl dgst -sha256 -hmac "$SIGNATURE_KEY" \\
  | awk '{print $2}')

curl -X POST \\
  -H "X-Signature-256: sha256=$SIGNATURE" \\
  -H "Content-Type: application/json" \\
  --data "$PAYLOAD" \\
  ${webhookUrl}`;
        break;

      case "headerkey": {
        title = "Header Key Authentication";
        description = "Use header key to authenticate your webhook requests.";
        const keyName = metadata?.headerKeyName || config?.headerKeyName || "X-API-Key";
        const keyValue = metadata?.headerKeyValue || config?.headerKeyValue || "<your-key>";
        code = `export HEADER_KEY="${keyValue}"
export PAYLOAD='{"hello":"world"}'

curl -X POST \\
  -H "${keyName}: $HEADER_KEY" \\
  -H "Content-Type: application/json" \\
  --data "$PAYLOAD" \\
  ${webhookUrl}`;
        break;
      }

      default:
        title = "No Authentication";
        description = "Send webhook requests without authentication.";
        code = `export PAYLOAD='{"hello":"world"}'

curl -X POST \\
  -H "Content-Type: application/json" \\
  --data "$PAYLOAD" \\
  ${webhookUrl}`;
        break;
    }

    return (
      <div className="border-t-1 border-gray-200">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{title}</span>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{description}</p>
            <div className="relative group mt-2">
              <pre className="text-sm text-gray-800 dark:text-gray-100 border-1 p-3 bg-gray-50 dark:bg-gray-800 rounded-md font-mono whitespace-pre overflow-x-auto">
                {code}
              </pre>
              <CopyCodeButton code={code} />
            </div>
          </div>
        </div>
      </div>
    );
  },
};
