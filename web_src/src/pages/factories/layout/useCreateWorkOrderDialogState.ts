import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { createWorkOrderPath, workOrderDetailPath, workOrdersPath } from "../lib/factoryPagePaths";

export function useCreateWorkOrderDialogState(organizationId: string, factoryKey: string, canCreate: boolean) {
  const navigate = useNavigate();
  const location = useLocation();
  const isCreateWorkOrderRoute = location.pathname === createWorkOrderPath(organizationId, factoryKey);
  const [createWorkOrderOpen, setCreateWorkOrderOpen] = useState(false);

  const openCreateWorkOrder = useCallback(() => {
    if (!canCreate) {
      return;
    }
    setCreateWorkOrderOpen(true);
  }, [canCreate]);

  useEffect(() => {
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

  return { createWorkOrderOpen, openCreateWorkOrder, closeCreateWorkOrder, completeCreateWorkOrder };
}
