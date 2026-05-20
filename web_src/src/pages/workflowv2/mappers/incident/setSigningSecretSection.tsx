import { useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import {
  canvasesDescribeCanvas,
  canvasesDescribeCanvasVersion,
  canvasesInvokeNodeTriggerHook,
  canvasesUpdateCanvasVersion,
  type CanvasesCanvas,
  type CanvasesCanvasVersion,
} from "@/api-client";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { canvasKeys } from "@/hooks/useCanvasData";

/** Applies signingSecretConfigured to the given node on a canvas copy. Does not mutate. */
function applySigningSecretConfigured(
  canvas: CanvasesCanvasVersion,
  nodeId: string,
  configured: boolean,
): CanvasesCanvasVersion | null {
  if (!canvas?.spec?.nodes) return null;
  const updatedNodes = canvas.spec.nodes.map((n) =>
    n.id === nodeId ? { ...n, configuration: { ...n.configuration, signingSecretConfigured: configured } } : n,
  );
  return { ...canvas, spec: { ...canvas.spec, nodes: updatedNodes } };
}

export function SetSigningSecretSection({ nodeId }: { nodeId: string }) {
  const [secret, setSecret] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const queryClient = useQueryClient();
  const { organizationId, canvasId } = useParams<{ organizationId: string; canvasId: string }>();
  const [searchParams] = useSearchParams();
  const versionId = searchParams.get("version");

  const handleSetSecretAndSave = async () => {
    if (!canvasId || !organizationId) return;
    if (!versionId) {
      showErrorToast("Create or select a version before saving signing secret.");
      return;
    }
    setIsSubmitting(true);
    setSuccess(false);
    try {
      const invokeResponse = await canvasesInvokeNodeTriggerHook(
        withOrganizationHeader({
          path: { canvasId, nodeId, hookName: "setSecret" },
          body: { parameters: { webhookSigningSecret: secret } },
        }),
      );
      const configured = (invokeResponse.data?.result?.signingSecretConfigured as boolean) ?? false;

      const freshVersion = await queryClient.fetchQuery({
        queryKey: canvasKeys.versionDetail(canvasId, versionId),
        queryFn: async () => {
          const response = await canvasesDescribeCanvasVersion(
            withOrganizationHeader({ path: { canvasId, versionId } }),
          );
          return response.data?.version;
        },
      });

      if (!freshVersion) {
        showErrorToast("Could not load version to save");
        return;
      }

      const updatedVersion = applySigningSecretConfigured(freshVersion, nodeId, configured);
      if (updatedVersion) {
        let liveCanvas = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
        if (!liveCanvas) {
          const canvasResponse = await canvasesDescribeCanvas(withOrganizationHeader({ path: { id: canvasId } }));
          liveCanvas = canvasResponse.data?.canvas;
        }
        if (!liveCanvas?.metadata?.name) {
          showErrorToast("Could not load canvas metadata for version update.");
          return;
        }

        await canvasesUpdateCanvasVersion(
          withOrganizationHeader({
            path: { canvasId, versionId },
            body: {
              canvas: {
                metadata: {
                  name: liveCanvas.metadata.name,
                  description: liveCanvas.metadata.description,
                },
                spec: { nodes: updatedVersion.spec?.nodes, edges: updatedVersion.spec?.edges },
              },
            },
          }),
        );
        queryClient.setQueryData(canvasKeys.versionDetail(canvasId, versionId), updatedVersion);
        setSuccess(true);
        setSecret("");
        showSuccessToast(
          configured ? "Signing secret set and version saved" : "Signing secret cleared and version saved",
        );
      } else {
        showErrorToast("Could not update version (invalid canvas structure).");
      }
    } catch {
      showErrorToast("Failed to set signing secret or save version");
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
          className="font-mono text-sm ph-no-capture"
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
}
