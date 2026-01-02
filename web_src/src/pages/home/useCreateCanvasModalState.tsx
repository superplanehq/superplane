import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useCreateWorkflow, useUpdateWorkflow } from "../../hooks/useWorkflowData";
import type { ComponentsEdge, ComponentsNode } from "@/api-client";

type ModalMode = "create" | "edit";

type WorkflowSummary = {
  id: string;
  name: string;
  description?: string;
  nodes?: ComponentsNode[];
  edges?: ComponentsEdge[];
};

export function useCreateCanvasModalState() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();

  const [modalState, setModalState] = useState<{ mode: ModalMode; workflow?: WorkflowSummary } | null>(null);

  const onOpen = () => setModalState({ mode: "create" });
  const onOpenEdit = (workflow: WorkflowSummary) => setModalState({ mode: "edit", workflow });
  const onClose = () => setModalState(null);

  const createMutation = useCreateWorkflow(organizationId || "");
  const updateMutation = useUpdateWorkflow(organizationId || "", modalState?.workflow?.id || "");

  const onSubmit = async (data: { name: string; description?: string }) => {
    if (!organizationId) {
      return;
    }

    if (modalState?.mode === "edit" && modalState.workflow?.id) {
      await updateMutation.mutateAsync({
        name: data.name,
        description: data.description,
        nodes: modalState.workflow?.nodes,
        edges: modalState.workflow?.edges,
      });
      onClose();
      return;
    }

    const result = await createMutation.mutateAsync({
      name: data.name,
      description: data.description,
    });

    if (result?.data?.workflow?.metadata?.id) {
      onClose();
      navigate(`/${organizationId}/workflows/${result.data.workflow.metadata.id}`);
    }
  };

  return {
    isOpen: modalState !== null,
    onOpen,
    onOpenEdit,
    onClose,
    onSubmit,
    isLoading: modalState?.mode === "edit" ? updateMutation.isPending : createMutation.isPending,
    initialData: modalState?.workflow
      ? { name: modalState.workflow.name, description: modalState.workflow.description }
      : undefined,
    mode: modalState?.mode ?? "create",
  };
}
