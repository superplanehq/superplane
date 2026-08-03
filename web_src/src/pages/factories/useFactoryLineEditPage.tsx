import { usePermissions } from "@/contexts/usePermissions";
import { useFactory, useFactoryApps } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useMemo } from "react";
import { factoryDetailPath } from "./factoryPagePaths";
import { isFactoryLineEditPageFailed, resolveFactoryLineEditRedirect } from "./factoryPageRedirects";
import { useFactoryLineEditActions } from "./useFactoryLineEditActions";

export function useFactoryLineEditPage(organizationId: string, factoryId: string, lineId?: string) {
  const { canAct } = usePermissions();
  const isCreate = !lineId;

  const { data: factory, isLoading: factoryLoading, error: factoryError } = useFactory(organizationId, factoryId);
  const { data: factoryApps = [], isLoading: appsLoading } = useFactoryApps(organizationId, factoryId);

  const line = useMemo(() => {
    if (isCreate || !lineId) {
      return null;
    }
    return factory?.lines?.find((entry) => entry.id === lineId) ?? null;
  }, [factory?.lines, isCreate, lineId]);

  const canUpdate = canAct("factories", "update");
  const factoryHref = factoryDetailPath(organizationId, factoryId);
  const actions = useFactoryLineEditActions(organizationId, factoryId, factoryHref, isCreate, lineId);

  usePageTitle([isCreate ? "New Line" : (line?.name ?? "Edit Line"), factory?.name ?? "Factory"]);

  useReportPageReady(!factoryLoading && Boolean(factory) && (isCreate || Boolean(line)), {
    failed: isFactoryLineEditPageFailed({ factoryError, isCreate, factoryLoading, line }),
  });

  const redirect = resolveFactoryLineEditRedirect({
    canUpdate,
    factoryLoading,
    factoryError,
    isCreate,
    factory,
    line,
    organizationId,
    factoryId,
  });
  if (redirect) {
    return { kind: "redirect" as const, element: redirect };
  }

  return {
    kind: "ready" as const,
    organizationId,
    factory,
    factoryHref,
    factoryApps,
    isLoading: factoryLoading || appsLoading,
    isCreate,
    line,
    ...actions,
  };
}
