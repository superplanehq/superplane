import { useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  canvasesDescribeCanvas,
  canvasesPutCanvasStaging,
  canvasesCommitCanvasStaging,
  canvasesInvokeNodeTriggerHook,
  type CanvasesCanvas,
  type CanvasesCanvasVersion,
} from "@/api-client";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { registerLocalStagingWrite } from "@/lib/canvasStagingEcho";
import { canvasKeys } from "@/hooks/useCanvasData";
import { useCanvasId } from "@/hooks/useCanvasId";
import { encodeRepositoryFileContent } from "../../files/lib/repository-files";
import { fetchCommittedCanvasVersionWithSpec } from "../../lib/repository-spec-files";
import { materializeCanvasSpec } from "../../lib/workflow-spec-files";
import { CANVAS_YAML_PATH } from "../../lib/workflow-spec-paths";

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

async function commitUpdatedCanvasVersionYaml(params: {
  canvasId: string;
  organizationId: string;
  versionId: string;
  updatedVersion: CanvasesCanvasVersion;
  queryClient: QueryClient;
}): Promise<boolean> {
  let liveCanvas = params.queryClient.getQueryData<CanvasesCanvas>(
    canvasKeys.detail(params.organizationId, params.canvasId),
  );
  if (!liveCanvas) {
    const canvasResponse = await canvasesDescribeCanvas(withOrganizationHeader({ path: { id: params.canvasId } }));
    liveCanvas = canvasResponse.data?.canvas;
  }
  if (!liveCanvas?.metadata?.name) {
    showErrorToast("Could not load canvas metadata for version update.");
    return false;
  }

  const canvasYaml = materializeCanvasSpec({
    ...liveCanvas,
    spec: params.updatedVersion.spec,
  });

  registerLocalStagingWrite(params.canvasId);
  await canvasesPutCanvasStaging(
    withOrganizationHeader({
      path: { canvasId: params.canvasId },
      body: {
        operations: [
          {
            path: CANVAS_YAML_PATH,
            content: encodeRepositoryFileContent(canvasYaml),
          },
        ],
      },
    }),
  );
  await canvasesCommitCanvasStaging(
    withOrganizationHeader({
      path: { canvasId: params.canvasId },
      body: { commitMessage: "Update signing secret configuration" },
    }),
  );
  params.queryClient.setQueryData(canvasKeys.versionDetail(params.canvasId, params.versionId), params.updatedVersion);
  return true;
}

export function SetSigningSecretSection({ nodeId }: { nodeId: string }) {
  const [secret, setSecret] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const queryClient = useQueryClient();
  const { organizationId } = useParams<{ organizationId: string }>();
  const canvasId = useCanvasId();
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
        queryFn: async () => fetchCommittedCanvasVersionWithSpec(canvasId, versionId),
      });

      if (!freshVersion) {
        showErrorToast("Could not load version to save");
        return;
      }

      const updatedVersion = applySigningSecretConfigured(freshVersion, nodeId, configured);
      if (!updatedVersion) {
        showErrorToast("Could not update version (invalid canvas structure).");
        return;
      }

      const committed = await commitUpdatedCanvasVersionYaml({
        canvasId,
        organizationId,
        versionId,
        updatedVersion,
        queryClient,
      });
      if (!committed) {
        return;
      }

      setSuccess(true);
      setSecret("");
      showSuccessToast(
        configured ? "Signing secret set and version saved" : "Signing secret cleared and version saved",
      );
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
