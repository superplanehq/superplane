import { useCallback } from "react";
import { useNavigate, useParams } from "react-router";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { PLACEHOLDER_NODE_CONTEXT_KEY, setAgentBootContext } from "@/lib/agentBootContext";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { writeCanvasRunsSidebarOpen } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { appPath } from "@/lib/appPaths";

interface UseCreateAppOptions {
  onCreated?: () => void;
}

function applyBlankAppBootContext(canvasId: string) {
  writeCanvasAgentSidebarOpen(canvasId, true);
  writeCanvasRunsSidebarOpen(canvasId, false);
  localStorage.setItem("canvasSidebarOpen", "false");
  setAgentBootContext(canvasId, "blank");
  sessionStorage.setItem(PLACEHOLDER_NODE_CONTEXT_KEY, canvasId);
}

export function useCreateApp({ onCreated }: UseCreateAppOptions = {}) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const { canAct } = usePermissions();
  const createCanvasMutation = useCreateCanvas(organizationId || "");
  const { mutateAsync: createCanvas } = createCanvasMutation;

  const canCreateCanvases = canAct("canvases", "create");
  const isSaving = createCanvasMutation.isPending;

  const createApp = useCallback(
    async (name: string) => {
      if (!organizationId || !canCreateCanvases || isSaving) {
        return;
      }

      try {
        const result = await createCanvas({
          name,
          method: "ui",
        });

        const canvasId = result?.data?.canvas?.metadata?.id;
        if (!canvasId) return;

        onCreated?.();
        applyBlankAppBootContext(canvasId);
        navigate(appPath(organizationId, canvasId, "?edit=1"));
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to create app"));
        throw error;
      }
    },
    [canCreateCanvases, createCanvas, isSaving, navigate, onCreated, organizationId],
  );

  return {
    createApp,
    isSaving,
  };
}
