import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { createWorkOrderPath, workOrdersPath } from "../lib/factoryPagePaths";

export function useCreateWorkOrderDialogState(organizationId: string, factoryId: string) {
  const navigate = useNavigate();
  const location = useLocation();
  const isCreateWorkOrderRoute = location.pathname === createWorkOrderPath(organizationId, factoryId);
  const [createWorkOrderOpen, setCreateWorkOrderOpen] = useState(false);
  const openCreateWorkOrder = useCallback(() => setCreateWorkOrderOpen(true), []);

  useEffect(() => {
    setCreateWorkOrderOpen(isCreateWorkOrderRoute);
  }, [factoryId, isCreateWorkOrderRoute]);

  const closeCreateWorkOrder = useCallback(() => {
    setCreateWorkOrderOpen(false);
    if (isCreateWorkOrderRoute) {
      navigate(workOrdersPath(organizationId, factoryId), { replace: true });
    }
  }, [factoryId, isCreateWorkOrderRoute, navigate, organizationId]);

  return { createWorkOrderOpen, openCreateWorkOrder, closeCreateWorkOrder };
}
