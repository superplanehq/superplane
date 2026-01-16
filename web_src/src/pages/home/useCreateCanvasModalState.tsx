import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useCreateWorkflow, useUpdateWorkflow, useWorkflowTemplates } from "../../hooks/useWorkflowData";
import type { ComponentsEdge, ComponentsNode } from "@/api-client";

type ModalMode = "create" | "edit";

type WorkflowSummary = {
  id: string;
  name: string;
  description?: string;
  nodes?: ComponentsNode[];
  edges?: ComponentsEdge[];
};

type WorkflowTemplateSummary = {
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
  const { data: workflowTemplates = [] } = useWorkflowTemplates(organizationId || "");

  const onSubmit = async (data: { name: string; description?: string; templateId?: string }) => {
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

    const selectedTemplate = workflowTemplates.find((template) => template.metadata?.id === data.templateId);
    const result = await createMutation.mutateAsync({
      name: data.name,
      description: data.description,
      nodes: selectedTemplate?.spec?.nodes,
      edges: selectedTemplate?.spec?.edges,
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
    templates: workflowTemplates
      .filter((template) => !!template.metadata?.id)
      .map((template) => ({
        id: template.metadata?.id || "",
        name: template.metadata?.name || "Untitled template",
        description: template.metadata?.description,
        nodes: template.spec?.nodes,
        edges: template.spec?.edges,
      })) as WorkflowTemplateSummary[],
    mode: modalState?.mode ?? "create",
  };
}
