import { useCreateWorkOrder, useDispatchWorkOrder } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useState } from "react";

const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 5000;

interface UseCreateWorkOrderComposerArgs {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreated: (orderId: string) => void;
}

export function useCreateWorkOrderComposer({
  organizationId,
  factoryId,
  onClose,
  onCreated,
}: UseCreateWorkOrderComposerArgs) {
  const createWorkOrder = useCreateWorkOrder(organizationId, factoryId);
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [assigneeIds, setAssigneeIds] = useState<string[]>([]);
  const [selectedLineName, setSelectedLineName] = useState("");
  const [titleError, setTitleError] = useState("");
  const [inFlightAction, setInFlightAction] = useState<"draft" | "send" | null>(null);

  const isSaving = inFlightAction !== null;
  const canSaveDraft = Boolean(title.trim()) && !isSaving;
  const canSendToLine = canSaveDraft && Boolean(selectedLineName);

  const goToOrder = (orderId: string | undefined) => {
    if (orderId) {
      onCreated(orderId);
      return;
    }
    onClose();
  };

  const saveOrder = async () => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setTitleError("Title is required");
      return null;
    }

    try {
      return await createWorkOrder.mutateAsync({
        title: trimmedTitle,
        description: description.trim(),
        assigneeIds,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create work order"));
      return null;
    }
  };

  const handleSaveDraft = async () => {
    setInFlightAction("draft");
    try {
      const order = await saveOrder();
      if (order) {
        goToOrder(order.id);
      }
    } finally {
      setInFlightAction(null);
    }
  };

  const handleSendToLine = async () => {
    if (!selectedLineName) {
      return;
    }

    setInFlightAction("send");
    try {
      const order = await saveOrder();
      if (!order?.id) {
        return;
      }

      try {
        await dispatchWorkOrder.mutateAsync({ orderId: order.id, lineName: selectedLineName });
        goToOrder(order.id);
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to send work order to line"));
        goToOrder(order.id);
      }
    } finally {
      setInFlightAction(null);
    }
  };

  const updateTitle = (next: string) => {
    setTitle(next.slice(0, MAX_TITLE_LENGTH));
    if (titleError) {
      setTitleError("");
    }
  };

  const updateDescription = (next: string) => {
    setDescription(next.slice(0, MAX_DESCRIPTION_LENGTH));
  };

  return {
    title,
    description,
    assigneeIds,
    selectedLineName,
    titleError,
    isSaving,
    isSavingDraft: inFlightAction === "draft",
    isSendingToLine: inFlightAction === "send",
    canSaveDraft,
    canSendToLine,
    maxDescriptionLength: MAX_DESCRIPTION_LENGTH,
    maxTitleLength: MAX_TITLE_LENGTH,
    setAssigneeIds,
    setSelectedLineName,
    updateTitle,
    updateDescription,
    handleSaveDraft,
    handleSendToLine,
  };
}
