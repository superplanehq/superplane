import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { canvasesDescribeCanvas, canvasesInvokeNodeTriggerAction, canvasesUpdateCanvas } from "@/api-client";
import type { CanvasesCanvas } from "@/api-client";
import { CustomFieldRenderer, NodeInfo } from "../types";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { canvasKeys } from "@/hooks/useCanvasData";

interface OnIncidentConfig {
  events?: string[];
  signingSecretConfigured?: boolean;
}

interface OnIncidentMetadata {
  webhookUrl?: string;
  signingSecretConfigured?: boolean;
}

const CopyWebhookUrlButton: React.FC<{ webhookUrl: string }> = ({ webhookUrl }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(webhookUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (_err) {
      showErrorToast("Failed to copy webhook URL");
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="inline-flex items-center gap-1.5 px-2 py-1 text-xs font-medium text-gray-700 dark:text-gray-200 border-1 border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-800"
      title={copied ? "Copied!" : "Copy webhook URL"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
      {copied ? "Copied" : "Copy"}
    </button>
  );
};

/** Applies signingSecretConfigured to the given node on a canvas copy. Does not mutate. */
function applySigningSecretConfigured(
  canvas: CanvasesCanvas,
  nodeId: string,
  configured: boolean,
): CanvasesCanvas | null {
  if (!canvas?.spec?.nodes) return null;
  const updatedNodes = canvas.spec.nodes.map((n) =>
    n.id === nodeId ? { ...n, configuration: { ...n.configuration, signingSecretConfigured: configured } } : n,
  );
  return { ...canvas, spec: { ...canvas.spec, nodes: updatedNodes } };
}

const SetSigningSecretSection: React.FC<{ nodeId: string }> = ({ nodeId }) => {
  const [secret, setSecret] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const queryClient = useQueryClient();
  const { organizationId, canvasId } = useParams<{ organizationId: string; canvasId: string }>();

  const handleSetSecretAndSave = async () => {
    if (!canvasId || !organizationId) return;
    setIsSubmitting(true);
    setSuccess(false);
    try {
      const invokeResponse = await canvasesInvokeNodeTriggerAction(
        withOrganizationHeader({
          path: { canvasId, nodeId, actionName: "setSecret" },
          body: { parameters: { webhookSigningSecret: secret } },
        }),
      );
      const configured = (invokeResponse.data?.result?.signingSecretConfigured as boolean) ?? false;

      // Refetch the canvas from the server so we save from the latest state instead of
      // the query cache, avoiding overwriting concurrent edits from other tabs or users.
      const freshCanvas = await queryClient.fetchQuery({
        queryKey: canvasKeys.detail(organizationId, canvasId),
        queryFn: async () => {
          const response = await canvasesDescribeCanvas(withOrganizationHeader({ path: { id: canvasId } }));
          return response.data?.canvas;
        },
      });

      if (!freshCanvas) {
        showErrorToast("Could not load canvas to save");
        return;
      }

      const updatedCanvas = applySigningSecretConfigured(freshCanvas, nodeId, configured);
      if (updatedCanvas) {
        await canvasesUpdateCanvas(
          withOrganizationHeader({
            path: { id: canvasId },
            body: {
              canvas: {
                metadata: updatedCanvas.metadata,
                spec: { nodes: updatedCanvas.spec?.nodes, edges: updatedCanvas.spec?.edges },
              },
            },
          }),
        );
        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedCanvas);
        setSuccess(true);
        setSecret("");
        showSuccessToast(
          configured ? "Signing secret set and canvas saved" : "Signing secret cleared and canvas saved",
        );
      } else {
        showErrorToast("Could not update canvas (invalid canvas structure). Try saving the canvas and try again.");
      }
    } catch (_err) {
      showErrorToast("Failed to set signing secret or save canvas");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-2">
      <Label htmlFor="incident-signing-secret" className="text-xs font-medium text-gray-700 dark:text-gray-300">
        Webhook signing secret
      </Label>
      <div className="flex items-center gap-2">
        <Input
          id="incident-signing-secret"
          type="password"
          autoComplete="off"
          placeholder="whsec_..."
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
          className="font-mono text-sm"
        />
        <Button
          type="button"
          size="default"
          onClick={handleSetSecretAndSave}
          disabled={isSubmitting}
          className="shrink-0 inline-flex items-center gap-2"
        >
          {isSubmitting ? (
            <>
              <Icon name="loader" size="sm" className="animate-spin" />
              Saving...
            </>
          ) : secret.trim() ? (
            "Set signing secret"
          ) : (
            "Clear signing secret"
          )}
        </Button>
      </div>
      {success && (
        <p className="text-xs text-green-600 dark:text-green-400">
          Signing secret saved. It is not stored in the workflow configuration.
        </p>
      )}
    </div>
  );
};

export const onIncidentCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const config = node.configuration as OnIncidentConfig | undefined;
    const metadata = node.metadata as OnIncidentMetadata | undefined;
    // Prefer config (persisted with canvas) so it works without workflow_nodes metadata merge
    const webhookConfigured = config?.signingSecretConfigured === true || metadata?.signingSecretConfigured === true;

    if (webhookConfigured) {
      return (
        <div className="border-t-1 border-gray-200 dark:border-gray-700 pt-4">
          <div className="text-xs text-gray-600 dark:text-gray-400 rounded-md border border-gray-200 dark:border-gray-600 px-3 py-2.5 bg-gray-50 dark:bg-gray-800/50">
            <p className="font-medium text-gray-700 dark:text-gray-300 mb-1">incident.io webhook is configured</p>
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
      <div className="border-t-1 border-gray-200 dark:border-gray-700 pt-4">
        <div className="space-y-3">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">incident.io Webhook Setup</span>
          <div className="text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md space-y-2">
            <div className="flex items-center justify-between gap-2">
              <span className="font-medium text-gray-700 dark:text-gray-200">URL for incident.io</span>
              {metadata?.webhookUrl && <CopyWebhookUrlButton webhookUrl={metadata.webhookUrl} />}
            </div>
            <pre className="mt-1 text-xs font-mono whitespace-pre-wrap break-all text-gray-800 dark:text-gray-100">
              {webhookUrl}
            </pre>
            <p className="text-gray-600 dark:text-gray-400">
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
