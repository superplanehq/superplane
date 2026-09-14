import type { FactoriesWorkOrder } from "@/api-client";
import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { createWorkOrderPath, factoryHomePath, workOrderDetailPath } from "../lib/factoryPagePaths";

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
    (orderNumber: string, order?: FactoriesWorkOrder) => {
      setCreateWorkOrderOpen(false);
      const lineId = lineIdFromPathname(location.pathname) ?? firstLineId;
      navigate(workOrderDetailPath(organizationId, factoryKey, orderNumber, lineId), {
        replace: isCreateWorkOrderRoute,
        state: order?.id ? { peekOrder: order } : undefined,
      });
    },
    [factoryKey, firstLineId, isCreateWorkOrderRoute, location.pathname, navigate, organizationId],
  );

  return { createWorkOrderOpen, openCreateWorkOrder, closeCreateWorkOrder, completeCreateWorkOrder };
}

function lineIdFromPathname(pathname: string): string | undefined {
  const match = /\/lines\/([^/]+)/.exec(pathname);
  const lineId = match?.[1];
  if (!lineId || lineId === "new") {
    return undefined;
  }
  return lineId;
}
