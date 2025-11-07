import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useCreateWorkflow } from "../../hooks/useWorkflowData";

export function useCreateCanvasModalState() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();

  const [isOpen, setIsOpen] = useState(false);

  const onOpen = () => setIsOpen(true);
  const onClose = () => setIsOpen(false);

  const mutation = useCreateWorkflow(organizationId || "");

  const isPending = mutation.isPending;

  const onSubmit = async (data: { name: string; description?: string }) => {
    if (!organizationId) return;

    const result = await mutation.mutateAsync({
      name: data.name,
      description: data.description,
    });

    if (result?.data?.workflow?.metadata?.id) {
      onClose();
      navigate(`/${organizationId}/workflows/${result.data.workflow.metadata.id}`);
    }
  };

  return {
    isOpen,
    onOpen,
    onClose,
    onSubmit,
    isPending,
  };
}
