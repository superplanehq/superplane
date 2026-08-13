import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { createWorkOrderPath, workOrderDetailPath, workOrdersPath } from "../lib/factoryPagePaths";

export function useCreateWorkOrderDialogState(organizationId: string, factoryId: string, canCreate: boolean) {
  const navigate = useNavigate();
  const location = useLocation();
  const isCreateWorkOrderRoute = location.pathname === createWorkOrderPath(organizationId, factoryId);
  const [createWorkOrderOpen, setCreateWorkOrderOpen] = useState(false);

  const openCreateWorkOrder = useCallback(() => {
    if (!canCreate) {
      return;
    }
    setCreateWorkOrderOpen(true);
  }, [canCreate]);

  useEffect(() => {
    setCreateWorkOrderOpen(canCreate && isCreateWorkOrderRoute);
  }, [canCreate, factoryId, isCreateWorkOrderRoute]);

  const closeCreateWorkOrder = useCallback(() => {
    setCreateWorkOrderOpen(false);
    if (isCreateWorkOrderRoute) {
      navigate(workOrdersPath(organizationId, factoryId), { replace: true });
    }
  }, [factoryId, isCreateWorkOrderRoute, navigate, organizationId]);

  const completeCreateWorkOrder = useCallback(
    (orderId: string) => {
      setCreateWorkOrderOpen(false);
      navigate(workOrderDetailPath(organizationId, factoryId, orderId), {
        replace: isCreateWorkOrderRoute,
      });
    },
    [factoryId, isCreateWorkOrderRoute, navigate, organizationId],
  );

  return { createWorkOrderOpen, openCreateWorkOrder, closeCreateWorkOrder, completeCreateWorkOrder };
}
