import { useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas, useUpdateCanvasFolderMembership } from "@/hooks/useCanvasData";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { PLACEHOLDER_NODE_CONTEXT_KEY, setAgentBootContext } from "@/lib/agentBootContext";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { writeCanvasRunsSidebarOpen } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { appPath } from "@/lib/appPaths";
import { appendCanvasToFolderMembership } from "./canvasFolderMembership";
import type { CanvasFolderData } from "./types";

interface UseCreateAppOptions {
  folder?: CanvasFolderData;
  onCreated?: () => void;
}

export function useCreateApp({ folder, onCreated }: UseCreateAppOptions = {}) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const { canAct } = usePermissions();
  const createCanvasMutation = useCreateCanvas(organizationId || "");
  const updateCanvasFolderMembershipMutation = useUpdateCanvasFolderMembership(organizationId || "");
  const { mutateAsync: createCanvas } = createCanvasMutation;
  const { mutateAsync: updateCanvasFolderMembership } = updateCanvasFolderMembershipMutation;

  const canCreateCanvases = canAct("canvases", "create");
  const isSaving = createCanvasMutation.isPending || updateCanvasFolderMembershipMutation.isPending;

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
        if (canvasId) {
          if (folder) {
            try {
              await updateCanvasFolderMembership(appendCanvasToFolderMembership(folder, canvasId));
            } catch (error) {
              showErrorToast(getApiErrorMessage(error, "App created, but failed to add it to folder"));
              return;
            }
          }

          onCreated?.();
          // A new app always starts with the agent panel open (stored per canvas).
          writeCanvasAgentSidebarOpen(canvasId, true);
          writeCanvasRunsSidebarOpen(canvasId, false);
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
    [
      canCreateCanvases,
      createCanvas,
      folder,
      isSaving,
      navigate,
      onCreated,
      organizationId,
      updateCanvasFolderMembership,
    ],
  );

  return {
    createApp,
    isSaving,
  };
}
