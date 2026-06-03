import { useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import {
  canvasesCommitCanvasRepositoryFiles,
  canvasesDescribeCanvas,
  canvasesInvokeNodeTriggerHook,
  canvasesListDraftBranches,
  type CanvasesCanvas,
} from "@/api-client";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { CANVAS_YAML_PATH } from "@/lib/canvas-staging";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { canvasKeys } from "@/hooks/useCanvasData";
import { buildCanvasYamlFromWorkflow, parseCanvasYamlToSpec } from "../../lib/canvas-yaml-staging";
import { encodeRepositoryFileContent, fetchCanvasRepositoryFileContent } from "../../lib/canvas-repository-files";

function applySigningSecretConfigured(
  canvas: CanvasesCanvas,
  nodeId: string,
  configured: boolean,
): CanvasesCanvas | null {
  if (!canvas?.spec?.nodes) return null;
  const updatedNodes = canvas.spec.nodes.map((node) =>
    node.id === nodeId
      ? { ...node, configuration: { ...node.configuration, signingSecretConfigured: configured } }
      : node,
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
  const branch = searchParams.get("branch");

  const handleSetSecretAndSave = async () => {
    if (!canvasId || !organizationId) return;
    if (!branch) {
      showErrorToast("Start editing a draft branch before saving signing secret.");
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

      let liveCanvas = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
      if (!liveCanvas) {
        const canvasResponse = await canvasesDescribeCanvas(withOrganizationHeader({ path: { id: canvasId } }));
        liveCanvas = canvasResponse.data?.canvas;
      }
      if (!liveCanvas?.metadata?.name) {
        showErrorToast("Could not load canvas metadata for commit.");
        return;
      }

      const yamlText = await fetchCanvasRepositoryFileContent(canvasId, CANVAS_YAML_PATH, branch);
      const branchSpec = parseCanvasYamlToSpec(yamlText);
      const branchCanvas: CanvasesCanvas = {
        ...liveCanvas,
        spec: branchSpec ?? liveCanvas.spec,
      };

      const updatedCanvas = applySigningSecretConfigured(branchCanvas, nodeId, configured);
      if (!updatedCanvas) {
        showErrorToast("Could not update canvas (invalid canvas structure).");
        return;
      }

      const draftsResponse = await canvasesListDraftBranches(withOrganizationHeader({ path: { canvasId } }));
      const activeDraft = draftsResponse.data?.branches?.find((draft) => draft.branchName === branch);

      await canvasesCommitCanvasRepositoryFiles(
        withOrganizationHeader({
          path: { canvasId },
          body: {
            message: "Update signing secret configuration",
            branch,
            expectedHeadSha: activeDraft?.tipSha,
            operations: [
              {
                path: CANVAS_YAML_PATH,
                content: encodeRepositoryFileContent(buildCanvasYamlFromWorkflow(updatedCanvas)),
              },
            ],
          },
        }),
      );

      await queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(canvasId) });
      await queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFiles(canvasId, branch) });

      setSuccess(true);
      setSecret("");
      showSuccessToast(
        configured
          ? "Signing secret set and committed to draft branch"
          : "Signing secret cleared and committed to draft branch",
      );
    } catch {
      showErrorToast("Failed to set signing secret or commit changes");
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
