import { useCreateCanvas } from "@/hooks/useCanvasData";
import { factoryAppsKey, useDispatchWorkOrder } from "@/hooks/useFactoryData";
import { appPath } from "@/lib/appPaths";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

export function useFactoryDetailActions(
  organizationId: string,
  factoryId: string,
  setCreateAppOpen: (open: boolean) => void,
) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const createCanvas = useCreateCanvas(organizationId);

  const handleCreateApp = async (input: { name: string; description: string }) => {
    try {
      const result = await createCanvas.mutateAsync({
        name: input.name,
        description: input.description,
        factoryId,
        method: "ui",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      setCreateAppOpen(false);
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      if (canvasId) {
        showSuccessToast("Factory app created.");
        navigate(appPath(organizationId, canvasId, "?edit=1"));
      }
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create factory app"));
      throw error;
    }
  };

  const handleDispatch = async (orderId: string, lineName: string) => {
    await dispatchWorkOrder.mutateAsync({ orderId, lineName });
    showSuccessToast(`Dispatched to ${lineName}.`);
  };

  return {
    handleCreateApp,
    handleDispatch,
    isCreateAppSaving: createCanvas.isPending,
    isDispatching: dispatchWorkOrder.isPending,
  };
}
