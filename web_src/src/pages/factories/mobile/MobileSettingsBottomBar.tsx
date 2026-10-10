import type { FactoriesFactory } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useMemo } from "react";

import { CreateWorkOrderDialog } from "../CreateWorkOrderDialog";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { useCreateWorkOrderDialogState } from "../layout/useCreateWorkOrderDialogState";
import { firstFactoryLineId } from "../lib/factoryPagePaths";
import { MobileBottomBar } from "./MobileBottomBar";

/**
 * Bottom bar of the phone workspace shell, for the settings layout. Settings
 * pages do not render inside `MobileFactoriesLayout`, so this supplies the
 * workspace layout context and the Create task dialog the bar depends on.
 */
export function MobileSettingsBottomBar({
  organizationId,
  factoryId,
  routeSegment,
  factory,
  factories,
}: {
  organizationId: string;
  factoryId: string;
  routeSegment: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
}) {
  const { canAct } = usePermissions();
  const canCreateWorkOrder = canAct("work_orders", "create");
  const { createWorkOrderOpen, openCreateWorkOrder, closeCreateWorkOrder, completeCreateWorkOrder } =
    useCreateWorkOrderDialogState(organizationId, routeSegment, canCreateWorkOrder, firstFactoryLineId(factory));

  const layoutContextValue = useMemo(
    () => ({
      organizationId,
      factoryId,
      factoryKey: factory.key ?? routeSegment,
      routeSegment,
      factory,
      factories,
      openCreateWorkOrder,
    }),
    [organizationId, factoryId, factory, routeSegment, factories, openCreateWorkOrder],
  );

  return (
    <FactoriesLayoutContext.Provider value={layoutContextValue}>
      <MobileBottomBar />
      {canCreateWorkOrder ? (
        <CreateWorkOrderDialog
          open={createWorkOrderOpen}
          onClose={closeCreateWorkOrder}
          onCreated={completeCreateWorkOrder}
        />
      ) : null}
    </FactoriesLayoutContext.Provider>
  );
}
