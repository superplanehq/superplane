import { writeCanvasRunsSidebarOpen } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { useCreateCanvas, useUpdateCanvasFolderMembership } from "@/hooks/useCanvasData";
import { PLACEHOLDER_NODE_CONTEXT_KEY, setAgentBootContext } from "@/lib/agentBootContext";
import { appPath } from "@/lib/appPaths";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useCallback } from "react";
import { useNavigate, type NavigateFunction } from "react-router-dom";
import type { CanvasFolderData } from "./types";

interface UseCreateAppInFolderOptions {
  folder: CanvasFolderData;
  organizationId: string;
  canCreateCanvases: boolean;
  canUpdateCanvases: boolean;
}

type CreateCanvas = ReturnType<typeof useCreateCanvas>["mutateAsync"];

export function useCreateAppInFolder({
  folder,
  organizationId,
  canCreateCanvases,
  canUpdateCanvases,
}: UseCreateAppInFolderOptions) {
  const navigate = useNavigate();
  const createCanvasMutation = useCreateCanvas(organizationId);
  const updateCanvasFolderMembershipMutation = useUpdateCanvasFolderMembership(organizationId);
  const { mutateAsync: createCanvas } = createCanvasMutation;
  const { mutateAsync: updateCanvasFolderMembership } = updateCanvasFolderMembershipMutation;
  const isCreatingAppInFolder = createCanvasMutation.isPending || updateCanvasFolderMembershipMutation.isPending;

  const createAppInFolder = useCallback(async () => {
    if (!organizationId || !canCreateCanvases || !canUpdateCanvases || isCreatingAppInFolder) {
      return;
    }

    const canvasId = await createBlankCanvas(createCanvas);
    if (!canvasId) {
      return;
    }

    try {
      await updateCanvasFolderMembership({
        folderId: folder.id,
        title: folder.title,
        backgroundColor: folder.backgroundColor,
        canvasIds: addCanvasId(folder.canvasIds, canvasId),
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "App created, but failed to add it to folder"));
      return;
    }

    openCreatedCanvas(organizationId, canvasId, navigate);
  }, [
    canCreateCanvases,
    canUpdateCanvases,
    createCanvas,
    folder,
    isCreatingAppInFolder,
    navigate,
    organizationId,
    updateCanvasFolderMembership,
  ]);

  return {
    createAppInFolder,
    isCreatingAppInFolder,
  };
}

async function createBlankCanvas(createCanvas: CreateCanvas) {
  try {
    const result = await createCanvas({ name: generateCanvasName(), method: "ui" });
    const canvasId = result?.data?.canvas?.metadata?.id;
    if (!canvasId) {
      showErrorToast("Failed to create app");
      return null;
    }

    return canvasId;
  } catch (error) {
    showErrorToast(getUsageLimitToastMessage(error, "Failed to create app"));
    return null;
  }
}

function addCanvasId(canvasIds: string[], canvasId: string) {
  return canvasIds.includes(canvasId) ? canvasIds : [...canvasIds, canvasId];
}

function openCreatedCanvas(organizationId: string, canvasId: string, navigate: NavigateFunction) {
  writeCanvasAgentSidebarOpen(canvasId, true);
  writeCanvasRunsSidebarOpen(canvasId, false);
  localStorage.setItem("canvasSidebarOpen", "false");
  setAgentBootContext(canvasId, "blank");
  sessionStorage.setItem(PLACEHOLDER_NODE_CONTEXT_KEY, canvasId);
  navigate(appPath(organizationId, canvasId, "?edit=1"));
}
