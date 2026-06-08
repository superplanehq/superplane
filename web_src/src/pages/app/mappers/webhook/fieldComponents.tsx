import { useState } from "react";
import React from "react";
import { canvasesInvokeNodeTriggerHook } from "@/api-client";
import { Icon } from "@/components/Icon";
import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { useCanvasId } from "@/hooks/useCanvasId";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { canvasKeys } from "@/hooks/useCanvasData";
import { showErrorToast } from "@/lib/toast";

export const CopyCodeButton: React.FC<{ code: string }> = ({ code }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
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

export const ResetAuthButton: React.FC<{
  nodeId: string;
  authMethod: string;
  onSuccess?: (newSecret: string) => void;
}> = ({ nodeId, authMethod, onSuccess }) => {
  const [isResetting, setIsResetting] = useState(false);
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const { organizationId } = useParams<{ organizationId: string }>();
  const canvasId = useCanvasId();

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
      const response = await canvasesInvokeNodeTriggerHook(
        withOrganizationHeader({
          path: {
            canvasId: canvasId,
            nodeId: nodeId,
            hookName: "resetAuthentication",
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

        if (organizationId) {
          queryClient.invalidateQueries({
            queryKey: canvasKeys.detail(organizationId, canvasId),
          });
        }
      }
    } catch {
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
