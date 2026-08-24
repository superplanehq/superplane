import { useCreateWorkOrder, useDispatchWorkOrder } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useRef, useState } from "react";

const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 5000;

interface UseCreateWorkOrderComposerArgs {
  organizationId: string;
  factoryId: string;
  /** Title carried over from the surface that opened the composer. */
  initialTitle?: string;
  onClose: () => void;
  onCreated: (orderNumber: string) => void;
}

export function useCreateWorkOrderComposer({
  organizationId,
  factoryId,
  initialTitle = "",
  onClose,
  onCreated,
}: UseCreateWorkOrderComposerArgs) {
  const createWorkOrder = useCreateWorkOrder(organizationId, factoryId);
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const { data: me } = useMe(false, organizationId);

  const [title, setTitle] = useState(() => initialTitle.slice(0, MAX_TITLE_LENGTH));
  const [description, setDescription] = useState("");
  const [assigneeIds, setAssigneeIdsInternal] = useState<string[]>([]);
  const [titleError, setTitleError] = useState("");
  const [inFlightAction, setInFlightAction] = useState<"draft" | "send" | null>(null);
  const hasSeededOwner = useRef(false);

  const isSaving = inFlightAction !== null;
  const canSaveDraft = Boolean(title.trim()) && !isSaving;

  const setAssigneeIds = (ids: string[]) => {
    hasSeededOwner.current = true;
    setAssigneeIdsInternal(ids);
  };

  useEffect(() => {
    if (hasSeededOwner.current || !me?.id) {
      return;
    }
    hasSeededOwner.current = true;
    setAssigneeIdsInternal([me.id]);
  }, [me?.id]);

  const goToOrder = (order: { number?: string | number } | null) => {
    if (order?.number !== undefined && order.number !== "") {
      onCreated(String(order.number));
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
        goToOrder(order);
      }
    } finally {
      setInFlightAction(null);
    }
  };

  const handleSendToLine = async (lineName: string) => {
    if (!lineName) {
      return;
    }

    setInFlightAction("send");
    try {
      const order = await saveOrder();
      if (!order?.id) {
        return;
      }

      try {
        await dispatchWorkOrder.mutateAsync({ orderId: order.id, lineName });
        goToOrder(order);
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to send work order to line"));
        goToOrder(order);
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
    titleError,
    isSaving,
    isSavingDraft: inFlightAction === "draft",
    isSendingToLine: inFlightAction === "send",
    canSaveDraft,
    maxDescriptionLength: MAX_DESCRIPTION_LENGTH,
    maxTitleLength: MAX_TITLE_LENGTH,
    setAssigneeIds,
    updateTitle,
    updateDescription,
    handleSaveDraft,
    handleSendToLine,
  };
}
