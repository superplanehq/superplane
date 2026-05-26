import { useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas } from "@/hooks/useCanvasData";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast } from "@/lib/toast";

interface UseCreateAppOptions {
  onCreated?: () => void;
}

export function useCreateApp({ onCreated }: UseCreateAppOptions = {}) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
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

        const canvasId = result?.data?.canvas?.metadata?.id;
        if (canvasId) {
          onCreated?.();
          localStorage.setItem("canvasAgentSidebarOpen", "true");
          localStorage.setItem("canvasSidebarOpen", "false");
          sessionStorage.setItem("agent-boot-context", "blank");
          sessionStorage.setItem("add-placeholder-node", "1");
          navigate(`/${organizationId}/canvases/${canvasId}?edit=1`);
        }
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to create app"));
        throw error;
      }
    },
    [canCreateCanvases, createCanvasMutation, navigate, onCreated, organizationId],
  );

  return {
    createApp,
    isSaving: createCanvasMutation.isPending,
  };
}
