import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { createWorkOrderPath, factoryHomePath } from "../lib/factoryPagePaths";

export function useCreateWorkOrderDialogState(
  organizationId: string,
  factoryKey: string,
  canCreate: boolean,
  firstLineId?: string,
) {
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
      navigate(factoryHomePath(organizationId, factoryKey, firstLineId), { replace: true });
    }
  }, [factoryKey, firstLineId, isCreateWorkOrderRoute, navigate, organizationId]);

  const completeCreateWorkOrder = useCallback(
    (_orderNumber: string) => {
      setCreateWorkOrderOpen(false);
      navigate(factoryHomePath(organizationId, factoryKey, firstLineId), {
        replace: isCreateWorkOrderRoute,
      });
    },
    [factoryKey, firstLineId, isCreateWorkOrderRoute, navigate, organizationId],
  );

  return { createWorkOrderOpen, openCreateWorkOrder, closeCreateWorkOrder, completeCreateWorkOrder };
}
