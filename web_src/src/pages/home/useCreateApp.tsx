import { useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas } from "@/hooks/useCanvasData";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast } from "@/lib/toast";
import { PLACEHOLDER_NODE_CONTEXT_KEY, setAgentBootContext } from "@/lib/agentBootContext";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { appPath } from "@/lib/appPaths";

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
          // A new app always starts with the agent panel open (stored per canvas).
          writeCanvasAgentSidebarOpen(canvasId, true);
          localStorage.setItem("canvasSidebarOpen", "false");
          setAgentBootContext(canvasId, "blank");
          sessionStorage.setItem(PLACEHOLDER_NODE_CONTEXT_KEY, canvasId);
          navigate(appPath(organizationId, canvasId, "?edit=1"));
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
