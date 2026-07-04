import { useCallback, useState } from "react";
import { useParams } from "react-router-dom";
import { usePermissions } from "@/contexts/usePermissions";
import { useUpdateCanvas } from "@/hooks/useCanvasData";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast } from "@/lib/toast";
import type { CanvasCardData } from "./types";

export function useEditApp() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct } = usePermissions();
  const [editingCanvas, setEditingCanvas] = useState<CanvasCardData | null>(null);

  const updateCanvasMutation = useUpdateCanvas(organizationId || "", editingCanvas?.id || "");

  const canUpdateCanvases = canAct("canvases", "update");

  const openEdit = useCallback((canvas: CanvasCardData) => {
    setEditingCanvas(canvas);
  }, []);

  const closeEdit = useCallback(() => {
    setEditingCanvas(null);
  }, []);

  const saveApp = useCallback(
    async (data: { name: string; description: string }) => {
      if (!editingCanvas || !organizationId || !canUpdateCanvases || updateCanvasMutation.isPending) {
        return;
      }

      try {
        await updateCanvasMutation.mutateAsync({
          name: data.name,
          description: data.description,
        });
        closeEdit();
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to update app"));
        throw error;
      }
    },
    [canUpdateCanvases, closeEdit, editingCanvas, organizationId, updateCanvasMutation],
  );

  return {
    editingCanvas,
    openEdit,
    closeEdit,
    saveApp,
    isSaving: updateCanvasMutation.isPending,
    isOpen: editingCanvas !== null,
  };
}
