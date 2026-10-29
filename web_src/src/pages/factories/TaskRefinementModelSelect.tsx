import { useUpdateWorkOrder, useWorkOrder } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useFactoryTaskIdentity } from "./factoryTaskContext";
import { useWorkOrderRefinementModels } from "./useWorkOrderRefinementModels";
import { WorkOrderRefinementModelSelect } from "./WorkOrderRefinementModelSelect";

export function TaskRefinementModelSelect({ organizationId, canEdit }: { organizationId?: string; canEdit: boolean }) {
  const identity = useFactoryTaskIdentity();
  if (!organizationId || !identity.factoryId || !identity.orderId) return null;
  return (
    <SavedRefinementModel
      organizationId={organizationId}
      factoryId={identity.factoryId}
      orderId={identity.orderId}
      canEdit={canEdit}
    />
  );
}

function SavedRefinementModel({
  organizationId,
  factoryId,
  orderId,
  canEdit,
}: {
  organizationId: string;
  factoryId: string;
  orderId: string;
  canEdit: boolean;
}) {
  const order = useWorkOrder(organizationId, factoryId, orderId);
  const selection = useWorkOrderRefinementModels(organizationId, factoryId, orderId);
  const update = useUpdateWorkOrder(organizationId, factoryId);
  if (order.data?.state !== "STATE_DRAFT") return null;
  const model = order.data.refinementModel ?? "";
  const options = [{ value: "", label: "Auto" }, ...selection.options];
  if (model && !options.some((option) => option.value === model))
    options.push({ value: model, label: `${model} (unavailable)` });
  return (
    <WorkOrderRefinementModelSelect
      model={model}
      selection={{ ...selection, options }}
      disabled={!canEdit || update.isPending}
      onChange={(refinementModel) =>
        update.mutate(
          { orderId, refinementModel },
          { onError: (error) => showErrorToast(getApiErrorMessage(error, "Could not update the refinement model.")) },
        )
      }
    />
  );
}
