import { useCreateWorkOrder, useFactory } from "@/hooks/useFactoryData";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { factoryDetailPath } from "./factoryPagePaths";
import { resolveFactoryLoadRedirect } from "./factoryPageRedirects";

export function useCreateWorkOrderPage(organizationId: string, factoryId: string) {
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [assigneeIds, setAssigneeIds] = useState<string[]>([]);
  const [titleError, setTitleError] = useState("");

  const { data: factory, isLoading: factoryLoading, error: factoryError } = useFactory(organizationId, factoryId);
  const createWorkOrder = useCreateWorkOrder(organizationId, factoryId);
  const { data: users = [] } = useOrganizationUsers(organizationId);

  const selectedAssigneeLabels = useMemo(() => {
    const labelById = new Map(
      users
        .filter((user) => user.metadata?.id)
        .map((user) => [user.metadata!.id!, user.metadata?.email || user.spec?.displayName || user.metadata!.id!]),
    );

    return assigneeIds.map((id) => labelById.get(id)).filter((label): label is string => Boolean(label));
  }, [assigneeIds, users]);

  usePageTitle(["New Work Order", factory?.name ?? "Factory"]);

  useReportPageReady(!factoryLoading && Boolean(factory), {
    failed: Boolean(factoryError),
  });

  const redirect = resolveFactoryLoadRedirect(factoryLoading, factoryError, organizationId);
  if (redirect) {
    return { kind: "redirect" as const, element: redirect };
  }

  const factoryHref = factoryDetailPath(organizationId, factoryId);

  const handleCreate = async () => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setTitleError("Title is required");
      return;
    }

    try {
      const order = await createWorkOrder.mutateAsync({
        title: trimmedTitle,
        description: description.trim(),
        assigneeIds,
      });
      navigate(order.id ? `${factoryHref}/orders/${order.id}` : factoryHref);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create work order"));
    }
  };

  return {
    kind: "ready" as const,
    factoryHref,
    factory,
    factoryLoading,
    organizationId,
    title,
    description,
    assigneeIds,
    titleError,
    selectedAssigneeLabels,
    isCreating: createWorkOrder.isPending,
    setTitle,
    setDescription,
    setAssigneeIds,
    clearTitleError: () => setTitleError(""),
    handleCreate,
  };
}
