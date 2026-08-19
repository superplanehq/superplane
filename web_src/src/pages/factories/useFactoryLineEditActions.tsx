import { useCreateFactoryLine, useUpdateFactoryLine, type FactoryLineStep } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useNavigate } from "react-router";

export function useFactoryLineEditActions(
  organizationId: string,
  factoryId: string,
  factoryHref: string,
  isCreate: boolean,
  lineId?: string,
) {
  const navigate = useNavigate();
  const createFactoryLine = useCreateFactoryLine(organizationId, factoryId);
  const updateFactoryLine = useUpdateFactoryLine(organizationId, factoryId);

  const handleSave = async (input: { name: string; steps: FactoryLineStep[] }) => {
    try {
      if (isCreate) {
        await createFactoryLine.mutateAsync(input);
        showSuccessToast("Line created.");
      } else if (lineId) {
        await updateFactoryLine.mutateAsync({
          lineId,
          name: input.name,
          steps: input.steps,
        });
        showSuccessToast("Line updated.");
      }
      navigate(factoryHref);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, isCreate ? "Failed to create line" : "Failed to update line"));
      throw error;
    }
  };

  return {
    handleSave,
    navigateToFactory: () => navigate(factoryHref),
    isSaving: createFactoryLine.isPending || updateFactoryLine.isPending,
  };
}
