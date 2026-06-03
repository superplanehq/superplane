import { useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { usePermissions } from "@/contexts/usePermissions";
import { canvasKeys, useCreateCanvas } from "@/hooks/useCanvasData";
import { bootstrapBlankCanvasDraft } from "@/lib/bootstrapBlankCanvasDraft";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast } from "@/lib/toast";
import { setAgentBootContext } from "@/lib/agentBootContext";

interface UseCreateAppOptions {
  onCreated?: () => void;
}

export function useCreateApp({ onCreated }: UseCreateAppOptions = {}) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { canAct } = usePermissions();
  const createCanvasMutation = useCreateCanvas(organizationId || "");

  const canCreateCanvases = canAct("canvases", "create");

  const createApp = useCallback(
    async (name: string) => {
      if (!organizationId || !canCreateCanvases || createCanvasMutation.isPending) {
        return;
      }

      try {
        const result = await createCanvasMutation.mutateAsync({
          name,
          method: "ui",
        });

        const canvas = result?.data?.canvas;
        const canvasId = canvas?.metadata?.id;
        if (!canvas || !canvasId) {
          return;
        }

        onCreated?.();
        localStorage.setItem("canvasAgentSidebarOpen", "true");
        localStorage.setItem("canvasSidebarOpen", "false");
        setAgentBootContext(canvasId, "blank");

        const branchName = await bootstrapBlankCanvasDraft(canvas);
        await queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(canvasId) });

        const branchParam = encodeURIComponent(branchName);
        navigate(`/${organizationId}/canvases/${canvasId}?branch=${branchParam}`);
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to create app"));
        throw error;
      }
    },
    [canCreateCanvases, createCanvasMutation, navigate, onCreated, organizationId, queryClient],
  );

  return {
    createApp,
    isSaving: createCanvasMutation.isPending,
  };
}
