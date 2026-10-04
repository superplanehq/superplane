import type { FactoriesFactory } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactories, useFactory } from "@/hooks/useFactoryData";
import { useFactoryWebsocket } from "@/hooks/useFactoryWebsocket";
import { useMemo, type ReactNode } from "react";
import { Navigate, Outlet, useLocation, useParams } from "react-router";

import { CreateWorkOrderDialog } from "../CreateWorkOrderDialog";
import { FactoriesLayoutError, FactoriesLayoutLoading } from "../layout/FactoriesLayout";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { useCreateWorkOrderDialogState } from "../layout/useCreateWorkOrderDialogState";
import {
  factoryRouteNeedsCanonicalRedirect,
  factoryRouteSegment,
  replaceFactoryKeySegment,
  resolveFactoryByKey,
} from "../lib/factoryKeyResolution";
import { firstFactoryLineId } from "../lib/factoryPagePaths";
import { useFactoriesThemeClass } from "../lib/useFactoriesThemeClass";
import { MobileBottomBar } from "./MobileBottomBar";

/**
 * Phone shell for a workspace. No sidebar: pages fill the screen and a
 * bottom bar carries navigation and Create task. Provides the same layout
 * context as the desktop shell so shared cards, dialogs, and gates work.
 */
export function MobileFactoriesLayout({ children }: { children?: ReactNode }) {
  const { organizationId, factoryKey } = useParams<{ organizationId: string; factoryKey: string }>();
  if (!organizationId || !factoryKey) {
    return null;
  }
  return (
    <MobileFactoryResolver organizationId={organizationId} factoryKey={factoryKey}>
      {children}
    </MobileFactoryResolver>
  );
}

function MobileFactoryResolver({
  organizationId,
  factoryKey,
  children,
}: {
  organizationId: string;
  factoryKey: string;
  children?: ReactNode;
}) {
  const location = useLocation();
  const { data: factories = [], isLoading, isFetching } = useFactories(organizationId);
  const resolution = resolveFactoryByKey(factories, factoryKey, isLoading || isFetching);

  if (factoryRouteNeedsCanonicalRedirect(resolution, factoryKey)) {
    const target = replaceFactoryKeySegment(
      location.pathname,
      organizationId,
      factoryKey,
      factoryRouteSegment(resolution.factory),
    );
    return <Navigate to={`${target}${location.search}`} replace />;
  }
  if (resolution.status === "not-found") {
    return <FactoriesLayoutError organizationId={organizationId} />;
  }
  if (resolution.status === "loading" || !resolution.factory?.id) {
    return <FactoriesLayoutLoading />;
  }
  return (
    <MobileFactoryShell
      organizationId={organizationId}
      factoryId={resolution.factory.id}
      factoryKey={resolution.factory.key ?? factoryKey}
      factories={factories}
    >
      {children}
    </MobileFactoryShell>
  );
}

function MobileFactoryShell({
  organizationId,
  factoryId,
  factoryKey,
  factories,
  children,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factories: FactoriesFactory[];
  children?: ReactNode;
}) {
  useFactoriesThemeClass();
  const { canAct } = usePermissions();
  const { data: describedFactory, error: factoryError } = useFactory(organizationId, factoryId);
  const factory = describedFactory ?? factories.find((item) => item.id === factoryId) ?? null;
  const routeSegment = factoryRouteSegment(factory) || factoryKey;
  const canCreateWorkOrder = canAct("work_orders", "create");
  const { createWorkOrderOpen, openCreateWorkOrder, closeCreateWorkOrder, completeCreateWorkOrder } =
    useCreateWorkOrderDialogState(organizationId, routeSegment, canCreateWorkOrder, firstFactoryLineId(factory));
  useFactoryWebsocket(organizationId, factoryId);

  const layoutContextValue = useMemo(
    () => ({
      organizationId,
      factoryId,
      factoryKey: factory?.key ?? factoryKey,
      routeSegment,
      factory,
      factories,
      openCreateWorkOrder,
    }),
    [organizationId, factoryId, factory, factoryKey, routeSegment, factories, openCreateWorkOrder],
  );

  if (factoryError) {
    return <FactoriesLayoutError organizationId={organizationId} />;
  }
  if (!factory) {
    return <FactoriesLayoutLoading />;
  }

  return (
    <FactoriesLayoutContext.Provider value={layoutContextValue}>
      <div
        className="flex h-dvh w-full flex-col bg-background text-foreground [--workspace-navigation-width:0px]"
        data-testid="mobile-factories-layout"
      >
        <main className="relative min-h-0 min-w-0 flex-1 overflow-y-auto">{children ?? <Outlet />}</main>
        <MobileBottomBar canCreateWorkOrder={canCreateWorkOrder} />
      </div>
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
