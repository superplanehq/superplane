import { useCreateWorkOrder, useDispatchWorkOrder } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";

import { workOrderDetailPath, workOrdersPath } from "./lib/factoryPagePaths";

const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 5000;

interface UseCreateWorkOrderComposerArgs {
  organizationId: string;
  factoryId: string;
  open: boolean;
  onClose: () => void;
}

export function useCreateWorkOrderComposer({
  organizationId,
  factoryId,
  open,
  onClose,
}: UseCreateWorkOrderComposerArgs) {
  const navigate = useNavigate();
  const createWorkOrder = useCreateWorkOrder(organizationId, factoryId);
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [assigneeIds, setAssigneeIds] = useState<string[]>([]);
  const [selectedLineName, setSelectedLineName] = useState("");
  const [titleError, setTitleError] = useState("");

  useEffect(() => {
    if (!open) {
      return;
    }
    setTitle("");
    setDescription("");
    setAssigneeIds([]);
    setSelectedLineName("");
    setTitleError("");
  }, [open]);

  const isSaving = createWorkOrder.isPending || dispatchWorkOrder.isPending;
  const canSaveDraft = Boolean(title.trim()) && !isSaving;
  const canSendToLine = canSaveDraft && Boolean(selectedLineName);

  const goToOrder = (orderId: string | undefined) => {
    onClose();
    const path = orderId
      ? workOrderDetailPath(organizationId, factoryId, orderId)
      : workOrdersPath(organizationId, factoryId);
    navigate(path);
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
    const order = await saveOrder();
    if (order) {
      goToOrder(order.id);
    }
  };

  const handleSendToLine = async () => {
    if (!selectedLineName) {
      return;
    }

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
  };

  const updateTitle = (next: string) => {
    if (next.length > MAX_TITLE_LENGTH) {
      return;
    }
    setTitle(next);
    if (titleError) {
      setTitleError("");
    }
  };

  const updateDescription = (next: string) => {
    if (next.length > MAX_DESCRIPTION_LENGTH) {
      return;
    }
    setDescription(next);
  };

  return {
    title,
    description,
    assigneeIds,
    selectedLineName,
    titleError,
    isSaving,
    isSavingDraft: createWorkOrder.isPending && !dispatchWorkOrder.isPending,
    isSendingToLine: dispatchWorkOrder.isPending,
    canSaveDraft,
    canSendToLine,
    maxDescriptionLength: MAX_DESCRIPTION_LENGTH,
    setAssigneeIds,
    setSelectedLineName,
    updateTitle,
    updateDescription,
    handleSaveDraft,
    handleSendToLine,
  };
}
