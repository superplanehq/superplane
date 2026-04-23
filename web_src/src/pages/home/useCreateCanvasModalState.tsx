import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useCreateCanvas, useCanvasTemplates } from "../../hooks/useCanvasData";
import type { SuperplaneComponentsEdge, SuperplaneComponentsNode } from "@/api-client";

type WorkflowTemplateSummary = {
  id: string;
  name: string;
  description?: string;
  nodes?: SuperplaneComponentsNode[];
  edges?: SuperplaneComponentsEdge[];
};

export function useCreateCanvasModalState() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();

  const [isOpen, setIsOpen] = useState(false);

  const onOpen = () => setIsOpen(true);
  const onClose = () => setIsOpen(false);

  const createMutation = useCreateCanvas(organizationId || "");
  const { data: workflowTemplates = [] } = useCanvasTemplates(organizationId || "");

  const onSubmit = async (data: { name: string; description?: string; templateId?: string }) => {
    if (!organizationId) {
      return;
    }

    const selectedTemplate = workflowTemplates.find((template) => template.metadata?.id === data.templateId);
    const result = await createMutation.mutateAsync({
      name: data.name,
      description: data.description,
      nodes: selectedTemplate?.spec?.nodes,
      edges: selectedTemplate?.spec?.edges,
    });

    if (result?.data?.canvas?.metadata?.id) {
      onClose();
      navigate(`/${organizationId}/canvases/${result.data.canvas.metadata.id}`);
    }
  };

  return {
    isOpen,
    onOpen,
    onClose,
    onSubmit,
    isLoading: createMutation.isPending,
    organizationId,
    templates: workflowTemplates
      .filter((template) => !!template.metadata?.id)
      .map((template) => ({
        id: template.metadata?.id || "",
        name: template.metadata?.name || "Untitled template",
        description: template.metadata?.description,
        nodes: template.spec?.nodes,
        edges: template.spec?.edges,
      })) as WorkflowTemplateSummary[],
  };
}
