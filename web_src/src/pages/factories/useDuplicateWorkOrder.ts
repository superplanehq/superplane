import type { FactoriesWorkOrder } from "@/api-client";
import { useCreateWorkOrder } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useNavigate } from "react-router";

import { duplicateWorkOrderCreateInput, type DuplicateWorkOrderSource } from "./lib/duplicateWorkOrderCreateInput";
import { workOrderDetailPath } from "./lib/factoryPagePaths";
import { canonicalWorkOrderNumber } from "./lib/workOrderNumberResolution";

const DUPLICATE_FAILED = "Failed to duplicate the task.";

export function useDuplicateWorkOrder(args: {
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  lineId?: string | null;
}) {
  const navigate = useNavigate();
  const createWorkOrder = useCreateWorkOrder(args.organizationId ?? "", args.factoryId ?? "");

  const duplicate = async (source: DuplicateWorkOrderSource) => {
    if (createWorkOrder.isPending) {
      return;
    }
    if (!args.organizationId || !args.factoryId || !args.factoryKey) {
      showErrorToast(DUPLICATE_FAILED);
      return;
    }

    try {
      const order = await createWorkOrder.mutateAsync(duplicateWorkOrderCreateInput(source));
      showSuccessToast("Task duplicated.");
      openDuplicatedWorkOrder(navigate, args, order);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, DUPLICATE_FAILED));
    }
  };

  return { duplicate, isPending: createWorkOrder.isPending };
}

function openDuplicatedWorkOrder(
  navigate: ReturnType<typeof useNavigate>,
  args: { organizationId?: string; factoryKey?: string; lineId?: string | null },
  order: FactoriesWorkOrder,
) {
  const number = canonicalWorkOrderNumber(order);
  if (!number || !args.organizationId || !args.factoryKey) {
    return;
  }
  navigate(workOrderDetailPath(args.organizationId, args.factoryKey, number, args.lineId), {
    state: order.id ? { peekOrder: order } : undefined,
  });
}
