import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";
import {
  useAddWorkOrderComment,
  useAnswerWorkOrderSurvey,
  useCloseWorkOrder,
  useDispatchWorkOrder,
  useUpdateWorkOrderAssignees,
  useUpdateWorkOrderStatus,
} from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import type { WorkOrderSurveyAnswerInput } from "./lib/workOrderSurvey";
import { formatWorkOrderResult, formatWorkOrderState } from "./lib/workOrderPresentation";

export function useWorkOrderDetailActions(organizationId: string, factoryId: string, orderId: string) {
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const closeWorkOrder = useCloseWorkOrder(organizationId, factoryId);
  const updateAssignees = useUpdateWorkOrderAssignees(organizationId, factoryId);
  const updateStatus = useUpdateWorkOrderStatus(organizationId, factoryId);
  const addComment = useAddWorkOrderComment(organizationId, factoryId);
  const answerSurvey = useAnswerWorkOrderSurvey(organizationId, factoryId);

  const handleAssigneesSave = async (nextAssigneeIds: string[]) => {
    try {
      await updateAssignees.mutateAsync({ orderId, assigneeIds: nextAssigneeIds });
      showSuccessToast("Owner updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update the owner"));
      throw error;
    }
  };

  const handleDispatch = async ({ lineName }: { lineName: string }) => {
    await dispatchWorkOrder.mutateAsync({ orderId, lineName });
    showSuccessToast(`Dispatched to ${lineName}.`);
  };

  const handleClose = async (result: FactoriesWorkOrderResult) => {
    try {
      await closeWorkOrder.mutateAsync({ orderId, result });
      showSuccessToast(`Task closed as ${formatWorkOrderResult(result).toLowerCase()}.`);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to close task"));
    }
  };

  const handleStatusChange = async (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => {
    try {
      await updateStatus.mutateAsync({ orderId, state, result });
      showSuccessToast(`Task moved to ${formatWorkOrderState(state).toLowerCase()}.`);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update status"));
      throw error;
    }
  };

  const handleAddComment = async (body: string, mentionedUserIds: string[]) => {
    try {
      await addComment.mutateAsync({ orderId, body, mentionedUserIds });
      showSuccessToast("Comment added.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to add comment"));
      throw error;
    }
  };

  const handleAnswerSurvey = async (answers: WorkOrderSurveyAnswerInput[]) => {
    try {
      await answerSurvey.mutateAsync({ orderId, answers });
      showSuccessToast("Answers sent to the agent.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to submit the survey"));
      throw error;
    }
  };

  const isCompleting = closeWorkOrder.isPending && closeWorkOrder.variables?.result === "RESULT_COMPLETED";
  const isRejecting = closeWorkOrder.isPending && closeWorkOrder.variables?.result === "RESULT_REJECTED";

  return {
    handleAssigneesSave,
    handleDispatch,
    handleClose,
    handleStatusChange,
    handleAddComment,
    handleAnswerSurvey,
    isDispatching: dispatchWorkOrder.isPending,
    isCompleting,
    isRejecting,
    isClosing: closeWorkOrder.isPending,
    isAssigneesSaving: updateAssignees.isPending,
    isUpdatingStatus: updateStatus.isPending,
    isAddingComment: addComment.isPending,
    isAnsweringSurvey: answerSurvey.isPending,
  };
}
