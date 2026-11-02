import { useUpdateWorkOrder, useWorkOrder } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useFactoryTaskIdentity } from "./factoryTaskContext";
import { useWorkOrderRefinementModels } from "./useWorkOrderRefinementModels";
import { WorkOrderRefinementModelSelect } from "./WorkOrderRefinementModelSelect";

type TaskRefinementModelSelectProps = {
  organizationId?: string;
  canEdit: boolean;
  allowReselect?: boolean;
  onSaveStart?: () => void;
  onError?: (message: string) => void;
};

export function TaskRefinementModelSelect({ organizationId, ...props }: TaskRefinementModelSelectProps) {
  const identity = useFactoryTaskIdentity();
  if (!organizationId || !identity.factoryId || !identity.orderId) return null;
  return (
    <SavedRefinementModel
      organizationId={organizationId}
      factoryId={identity.factoryId}
      orderId={identity.orderId}
      {...props}
    />
  );
}

function SavedRefinementModel({
  organizationId,
  factoryId,
  orderId,
  canEdit,
  allowReselect = false,
  onSaveStart,
  onError,
}: {
  organizationId: string;
  factoryId: string;
  orderId: string;
} & Omit<TaskRefinementModelSelectProps, "organizationId">) {
  const order = useWorkOrder(organizationId, factoryId, orderId);
  const selection = useWorkOrderRefinementModels(organizationId, factoryId, orderId);
  const update = useUpdateWorkOrder(organizationId, factoryId);
  if (order.data?.state !== "STATE_DRAFT") return null;
  const model = order.data.refinementModel ?? "";
  const options: typeof selection.options = [{ value: "", label: "Auto" }, ...selection.options];
  if (model && !options.some((option) => option.value === model))
    options.push({ value: model, label: `${model} (unavailable)`, disabled: true });
  return (
    <WorkOrderRefinementModelSelect
      model={model}
      selection={{ ...selection, options }}
      disabled={!canEdit || update.isPending}
      onChange={(refinementModel) => {
        if (refinementModel === model && !allowReselect) return;
        onSaveStart?.();
        update.mutate(
          { orderId, refinementModel },
          {
            onError: (error) => {
              const message = getApiErrorMessage(error, "Could not update the refinement model.");
              if (onError) onError(message);
              else showErrorToast(message);
            },
          },
        );
      }}
    />
  );
}
