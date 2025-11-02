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

    console.log("Creating workflow with data:", JSON.stringify(data, null, 2));

    const result = await mutation.mutateAsync({
      name: data.name,
      description: data.description,
    });

    console.log(JSON.stringify(result, null, 2));
    if (result?.data?.workflow?.id) {
      onClose();
      console.log("Navigating to workflow:", result.data.workflow.id);
      navigate(`/${organizationId}/workflows/${result.data.workflow.id}`);
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
