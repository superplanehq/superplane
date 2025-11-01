import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useCreateBlueprint } from "../../hooks/useBlueprintData";

export function useCreateCustomComponentModalState() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();

  const [isOpen, setIsOpen] = useState(false);

  const onOpen = () => setIsOpen(true);
  const onClose = () => setIsOpen(false);

  const mutation = useCreateBlueprint(organizationId || "");
  const isPending = mutation.isPending;

  const onSubmit = async (data: { name: string; description?: string }) => {
    if (!organizationId) return;

    const result = await mutation.mutateAsync({
      name: data.name,
      description: data.description,
    });

    if (result?.data?.blueprint?.id) {
      onClose();
      navigate(`/${organizationId}/custom-components/${result.data.blueprint.id}`);
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
