import type { FactoriesFactory } from "@/api-client";
import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactories, useFactory } from "@/hooks/useFactoryData";

import { resolveFactoryByKey } from "../../lib/factoryKeyResolution";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { FactoriesLayoutError, FactoriesLayoutLoading } from "../../layout/FactoriesLayout";
import { FactoriesLayoutContext } from "../../layout/factoriesLayoutContext";
import { OnboardingPage } from "./OnboardingPage";

const ignoreOpenCreateWorkOrder = () => undefined;

/** Renders the existing workspace wizard while the provisional organization stays outside the browser URL. */
export function InitialWorkspaceOnboarding({
  organizationId,
  factoryKey,
}: {
  organizationId: string;
  factoryKey: string;
}) {
  useFactoriesThemeClass();
  const factories = useFactories(organizationId);
  const resolution = resolveFactoryByKey(factories.data ?? [], factoryKey, factories.isLoading || factories.isFetching);

  if (factories.isError || resolution.status === "not-found") {
    return <FactoriesLayoutError organizationId={organizationId} />;
  }
  if (resolution.status === "loading" || !resolution.factory?.id) {
    return <FactoriesLayoutLoading />;
  }

  return (
    <PermissionsProvider organizationId={organizationId}>
      <ResolvedInitialWorkspaceOnboarding
        organizationId={organizationId}
        factoryId={resolution.factory.id}
        factoryKey={resolution.factory.key ?? factoryKey}
        factories={factories.data ?? []}
      />
    </PermissionsProvider>
  );
}

function ResolvedInitialWorkspaceOnboarding({
  organizationId,
  factoryId,
  factoryKey,
  factories,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factories: FactoriesFactory[];
}) {
  const factory = useFactory(organizationId, factoryId);
  const permissions = usePermissions();

  if (factory.isError) {
    return <FactoriesLayoutError organizationId={organizationId} />;
  }
  if (factory.isLoading || permissions.isLoading || !factory.data) {
    return <FactoriesLayoutLoading />;
  }

  return (
    <FactoriesLayoutContext.Provider
      value={{
        organizationId,
        factoryId,
        factoryKey,
        factory: factory.data,
        factories,
        openCreateWorkOrder: ignoreOpenCreateWorkOrder,
      }}
    >
      <main className="h-dvh overflow-y-auto bg-background">
        <OnboardingPage />
      </main>
    </FactoriesLayoutContext.Provider>
  );
}
