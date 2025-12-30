import { useState } from "react";
import {
  ComponentsNode,
  TriggersTrigger,
  WorkflowsWorkflowEvent,
  workflowsInvokeNodeTriggerAction,
} from "@/api-client";
import { getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer, CustomFieldRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";
import { Icon } from "@/components/Icon";
import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { workflowKeys } from "@/hooks/useWorkflowData";

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
  headerKeyName?: string;
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
  // Check for run_name in the webhook request body
  const runName = (event.data?.data as { body?: { run_name?: string } })?.body?.run_name;
  if (runName) {
    return `${runName}`;
  }

  // Fallback to method and date
  return `Webhook from ${event.nodeId}`;
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
  const { organizationId, workflowId } = useParams<{ organizationId: string; workflowId: string }>();

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
      case "headerkey":
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
    if (authMethod === "none" || !workflowId) return;

    setIsResetting(true);
    try {
      const response = await workflowsInvokeNodeTriggerAction(
        withOrganizationHeader({
          path: {
            workflowId: workflowId,
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
            queryKey: workflowKeys.detail(organizationId, workflowId),
          });
        }
      }
    } catch (error) {
      console.error("Failed to reset authentication:", error);
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
  render: (node: ComponentsNode, configuration: Record<string, unknown>) => {
    const metadata = node.metadata as WebhookMetadata | undefined;
    const config = configuration as WebhookConfiguration | undefined;
    const authMethod = config?.authentication || "none";
    const webhookUrl =
      metadata?.url ||
      "https://app.superplane.com/3ee1aa47-3a60-4c1f-b645-0b9859ab91f8/workflows/8d4dbb35-0034-499f-81da-226da05452e2/webhook/somestring-or-something";

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
          const keyName = config?.headerKeyName || "X-API-Key";
          const keyValue = secret || "<your-key>";
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

      return { title, description, code };
    };

    const { title, description, code } = generateCode(currentSecret as string);

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
          </div>
        </div>
      </div>
    );
  },
};
