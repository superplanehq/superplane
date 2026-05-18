import { useState } from "react";
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
import { CopyCodeButton, ResetAuthButton } from "./fieldComponents";

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

/**
 * Custom field renderer for webhook component configuration
 */
export const webhookCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as WebhookMetadata | undefined;
    const config = node.configuration as WebhookConfiguration | undefined;
    const authMethod = config?.authentication || "none";
    const headerName = config?.headerName || DEFAULT_HEADER_TOKEN_NAME;
    const signatureHeaderName = config?.signatureHeader?.trim() || DEFAULT_SIGNATURE_HEADER;
    const webhookUrl = metadata?.url || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    // State to track the currently displayed secret
    // eslint-disable-next-line react-hooks/rules-of-hooks
    const [currentSecret, setCurrentSecret] = useState<string | null>(null);

    const generateCode = (secret?: string) => {
      let description: string;
      let code: string;
      let title: string;
      let signatureKey: string;

      switch (authMethod) {
        case "signature":
          title = "HMAC Signature Authentication";
          description = "Use HMAC SHA-256 signature to authenticate your webhook requests.";
          signatureKey = secret || "<your-signature-key>";
          code = `export SIGNATURE_KEY="${signatureKey}"
export PAYLOAD='{"hello":"world"}'

export SIGNATURE=$(echo -n "$PAYLOAD" \\
  | openssl dgst -sha256 -hmac "$SIGNATURE_KEY" -binary \\
  | xxd -p -c 256)

curl -X POST \\
  -H "${signatureHeaderName}: sha256=$SIGNATURE" \\
  -H "Content-Type: application/json" \\
  --data-binary "$PAYLOAD" \\
  ${webhookUrl}`;
          break;

        case "bearer":
          title = "Bearer Token Authentication";
          description = "Use bearer token to authenticate your webhook requests.";
          signatureKey = secret || "<your-bearer-token>";
          code = `export BEARER_TOKEN="${signatureKey}"
export PAYLOAD='{"hello":"world"}'

curl -X POST \\
  -H "Authorization: Bearer $BEARER_TOKEN" \\
  -H "Content-Type: application/json" \\
  --data "$PAYLOAD" \\
  ${webhookUrl}`;
          break;

        case "header_token":
          title = "Header Token Authentication";
          description = `Use a raw token in the ${headerName} header to authenticate webhook requests.`;
          signatureKey = secret || "<your-header-token>";
          code = `export HEADER_TOKEN="${signatureKey}"
export PAYLOAD='{"hello":"world"}'

curl -X POST \\
  -H "${headerName}: $HEADER_TOKEN" \\
  -H "Content-Type: application/json" \\
  --data "$PAYLOAD" \\
  ${webhookUrl}`;
          break;

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

      return { title, description, code };
    };

    const { title, description, code } = generateCode(currentSecret as string);

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{title}</span>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{description}</p>

            {/* Webhook URL Copy Field */}
            <div className="mt-3">
              <label
                htmlFor="webhook-url-input"
                className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide"
              >
                Webhook URL
              </label>
              <div className="relative group mt-1">
                <input
                  id="webhook-url-input"
                  type="text"
                  value={webhookUrl}
                  readOnly
                  className="w-full text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono"
                />
                <CopyCodeButton code={webhookUrl} />
              </div>
            </div>

            <div className="relative group mt-3">
              <p className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Code Example
              </p>
              <div className="relative group mt-1">
                <pre className="text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono whitespace-pre overflow-x-auto">
                  {code}
                </pre>
                <CopyCodeButton code={code} />
              </div>
            </div>
            {metadata?.url ? (
              <ResetAuthButton
                nodeId={node.id!}
                authMethod={authMethod}
                onSuccess={(newSecret) => {
                  // Update the displayed code with the new secret
                  setCurrentSecret(newSecret);
                  // Auto-hide the secret from the code after 30 seconds for security
                  setTimeout(() => setCurrentSecret(null), 30000);
                }}
              />
            ) : (
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Save the canvas to generate a webhook URL and to be able of generating authentication secrets
              </p>
            )}
          </div>
        </div>
      </div>
    );
  },
};
