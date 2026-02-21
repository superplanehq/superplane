import { useState } from "react";
import { canvasesInvokeNodeTriggerAction } from "@/api-client";
import { getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer, CustomFieldRenderer, NodeInfo, TriggerRendererContext, TriggerEventContext } from "./types";
import { TriggerProps } from "@/ui/trigger";
import { Icon } from "@/components/Icon";
import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { canvasKeys } from "@/hooks/useCanvasData";
import { showErrorToast } from "@/utils/toast";

const DEFAULT_HEADER_TOKEN_NAME = "X-Webhook-Token";

interface WebhookConfiguration {
  authentication?: string;
  headerName?: string;
}

interface WebhookMetadata {
  url?: string;
  authentication?: string;
}

function formatAuthenticationMethod(auth: string, headerName?: string): string {
  switch (auth) {
    case "none":
      return "No authentication";
    case "signature":
      return "HMAC signature";
    case "bearer":
      return "Bearer token";
    case "header_token":
      return `Header token (${headerName || DEFAULT_HEADER_TOKEN_NAME})`;
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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    return {
      title: getWebhookEventTitle(context),
      subtitle: formatTimeAgo(new Date(context.event?.createdAt || "")),
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
          label: formatAuthenticationMethod(
            metadata?.authentication || configuration?.authentication || "none",
            configuration?.headerName,
          ),
        },
      ],
    };

    if (lastEvent) {
      const eventDate = new Date(lastEvent.createdAt);

      props.lastEventData = {
        title: getWebhookEventTitle({ event: lastEvent }),
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

/**
 * Reset authentication button component
 */
const ResetAuthButton: React.FC<{
  nodeId: string;
  authMethod: string;
  onSuccess?: (newSecret: string) => void;
}> = ({ nodeId, authMethod, onSuccess }) => {
  const [isResetting, setIsResetting] = useState(false);
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const { organizationId, canvasId } = useParams<{ organizationId: string; canvasId: string }>();

  const getAuthLabels = () => {
    switch (authMethod) {
      case "signature":
        return {
          buttonText: "Reset Signature Key",
          resettingText: "Resetting Signature Key...",
          successTitle: "New signature key generated",
          successDescription:
            "Please update your webhook client with the new signature key. This will only be shown once.",
        };
      case "bearer":
        return {
          buttonText: "Reset Bearer Token",
          resettingText: "Resetting Bearer Token...",
          successTitle: "New bearer token generated",
          successDescription:
            "Please update your webhook client with the new bearer token. This will only be shown once.",
        };
      case "header_token":
        return {
          buttonText: "Reset Header Token",
          resettingText: "Resetting Header Token...",
          successTitle: "New header token generated",
          successDescription:
            "Please update your webhook client with the new header token. This will only be shown once.",
        };
      default:
        return {
          buttonText: "Reset Authentication",
          resettingText: "Resetting...",
          successTitle: "New authentication secret generated",
          successDescription: "Please update your webhook client with the new secret. This will only be shown once.",
        };
    }
  };

  const labels = getAuthLabels();

  const handleResetAuth = async () => {
    if (authMethod === "none" || !canvasId) return;

    setIsResetting(true);
    try {
      const response = await canvasesInvokeNodeTriggerAction(
        withOrganizationHeader({
          path: {
            canvasId: canvasId,
            nodeId: nodeId,
            actionName: "resetAuthentication",
          },
          body: {
            parameters: {},
          },
        }),
      );

      const secret = response.data?.result?.secret as string | undefined;
      if (secret) {
        setNewSecret(secret);
        onSuccess?.(secret);

        // Invalidate workflow queries to refresh the UI
        if (organizationId) {
          queryClient.invalidateQueries({
            queryKey: canvasKeys.detail(organizationId, canvasId),
          });
        }
      }
    } catch (_error) {
      showErrorToast("Failed to reset authentication");
    } finally {
      setIsResetting(false);
    }
  };

  if (authMethod === "none") return null;

  return (
    <div className="mt-3 space-y-2">
      <div className="flex items-center gap-2">
        <button
          onClick={handleResetAuth}
          disabled={isResetting}
          className="inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-white bg-black hover:bg-gray-700 disabled:bg-gray-400 rounded-md transition-colors"
        >
          <Icon name={isResetting ? "loader" : "refresh-ccw"} size="sm" className={isResetting ? "animate-spin" : ""} />
          {isResetting ? labels.resettingText : labels.buttonText}
        </button>
      </div>

      {newSecret && (
        <div className="p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-md">
          <div className="flex items-start gap-2">
            <Icon name="triangle-alert" size="sm" className="text-yellow-600 dark:text-yellow-400 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">{labels.successTitle}</p>
              <p className="text-xs text-yellow-700 dark:text-yellow-300 mt-1">{labels.successDescription}</p>
              <div className="mt-2 relative group">
                <pre className="text-sm text-yellow-900 dark:text-yellow-100 bg-white dark:bg-gray-800 border border-yellow-300 dark:border-yellow-600 p-2 rounded font-mono break-all">
                  {newSecret}
                </pre>
                <CopyCodeButton code={newSecret} />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
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
  -H "X-Signature-256: sha256=$SIGNATURE" \\
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
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Webhook URL
              </label>
              <div className="relative group mt-1">
                <input
                  type="text"
                  value={webhookUrl}
                  readOnly
                  className="w-full text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono"
                />
                <CopyCodeButton code={webhookUrl} />
              </div>
            </div>

            <div className="relative group mt-3">
              <label className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">
                Code Example
              </label>
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
