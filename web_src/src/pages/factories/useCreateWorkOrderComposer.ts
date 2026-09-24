import type { FactoriesWorkOrder } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateWorkOrder } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useRef, useState } from "react";

import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";
import { derivedWorkOrderTitle } from "./lib/derivedWorkOrderTitle";

const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 5000;

export interface CreateWorkOrderComposerDraft {
  title: string;
  description: string;
}

interface UseCreateWorkOrderComposerArgs {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreated: (orderNumber: string, order?: FactoriesWorkOrder) => void;
}

export function useCreateWorkOrderComposer({
  organizationId,
  factoryId,
  onClose,
  onCreated,
}: UseCreateWorkOrderComposerArgs) {
  const createWorkOrder = useCreateWorkOrder(organizationId, factoryId);
  const { currentUserId } = usePermissions();

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [assigneeIds, setAssigneeIdsInternal] = useState<string[]>([]);
  const [titleError, setTitleError] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const hasSeededOwner = useRef(false);

  const canCreate = Boolean(title.trim()) && !isCreating;

  const setAssigneeIds = (ids: string[]) => {
    hasSeededOwner.current = true;
    setAssigneeIdsInternal(ids);
  };

  useEffect(() => {
    if (hasSeededOwner.current || !currentUserId) {
      return;
    }
    hasSeededOwner.current = true;
    setAssigneeIdsInternal([currentUserId]);
  }, [currentUserId]);

  const goToOrder = (order: FactoriesWorkOrder | null) => {
    if (order?.number !== undefined && order.number !== "") {
      onCreated(String(order.number), order);
      return;
    }
    onClose();
  };

  const handleCreate = async (draft?: CreateWorkOrderComposerDraft) => {
    const trimmedDescription = (draft?.description ?? description).trim();
    const trimmedTitle =
      (draft?.title ?? title).trim() ||
      (draft ? derivedWorkOrderTitle(trimmedDescription) || CREATE_WORK_ORDER_REQUEST_COPY.title : "");
    if (!trimmedTitle) {
      setTitleError("Title is required");
      return;
    }

    setIsCreating(true);
    try {
      const order = await createWorkOrder.mutateAsync({
        title: trimmedTitle,
        description: trimmedDescription,
        assigneeIds,
      });
      goToOrder(order);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create task"));
    } finally {
      setIsCreating(false);
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
    isCreating,
    canCreate,
    maxDescriptionLength: MAX_DESCRIPTION_LENGTH,
    maxTitleLength: MAX_TITLE_LENGTH,
    setAssigneeIds,
    updateTitle,
    updateDescription,
    handleCreate,
  };
}
