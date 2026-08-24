import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { createWorkOrderPath, workOrderDetailPath, workOrdersPath } from "../lib/factoryPagePaths";

export function useCreateWorkOrderDialogState(organizationId: string, factoryKey: string, canCreate: boolean) {
  const navigate = useNavigate();
  const location = useLocation();
  const isCreateWorkOrderRoute = location.pathname === createWorkOrderPath(organizationId, factoryKey);
  const [createWorkOrderOpen, setCreateWorkOrderOpen] = useState(false);
  const [createWorkOrderTitle, setCreateWorkOrderTitle] = useState("");

  const openCreateWorkOrder = useCallback(
    (initialTitle?: string) => {
      if (!canCreate) {
        return;
      }
      setCreateWorkOrderTitle(initialTitle ?? "");
      setCreateWorkOrderOpen(true);
    },
    [canCreate],
  );

  useEffect(() => {
    setCreateWorkOrderTitle("");
    setCreateWorkOrderOpen(canCreate && isCreateWorkOrderRoute);
  }, [canCreate, factoryKey, isCreateWorkOrderRoute, location.pathname]);

  const closeCreateWorkOrder = useCallback(() => {
    setCreateWorkOrderOpen(false);
    if (isCreateWorkOrderRoute) {
      navigate(workOrdersPath(organizationId, factoryKey), { replace: true });
    }
  }, [factoryKey, isCreateWorkOrderRoute, navigate, organizationId]);

  const completeCreateWorkOrder = useCallback(
    (orderNumber: string) => {
      setCreateWorkOrderOpen(false);
      navigate(workOrderDetailPath(organizationId, factoryKey, orderNumber), {
        replace: isCreateWorkOrderRoute,
      });
    },
    [factoryKey, isCreateWorkOrderRoute, navigate, organizationId],
  );

  return {
    createWorkOrderOpen,
    createWorkOrderTitle,
    openCreateWorkOrder,
    closeCreateWorkOrder,
    completeCreateWorkOrder,
  };
}
