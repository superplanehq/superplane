import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactories } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { Navigate, useParams } from "react-router";
import { factoryHomePath, firstFactoryLineId, newFactoryPath } from "./lib/factoryPagePaths";
import { pickReadyFactory, readLastVisitedFactory } from "./lib/lastVisitedFactory";
import { useFactoriesThemeClass } from "./lib/useFactoriesThemeClass";

export function FactoriesIndexPage() {
  const { organizationId } = useParams<{ organizationId: string }>();

  if (!organizationId) {
    return null;
  }

  return <FactoriesIndexPageContent organizationId={organizationId} />;
}

function FactoriesIndexPageContent({ organizationId }: { organizationId: string }) {
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: factories = [], isLoading, error } = useFactories(organizationId);

  useFactoriesThemeClass();
  usePageTitle(["Workspaces"]);

  if (isLoading) {
    return (
      <div
        className={cn("flex min-h-screen w-full items-center justify-center bg-gray-50", appDarkModeClasses.surface)}
      >
        <p className="text-sm text-gray-500 dark:text-gray-400">Loading workspaces…</p>
      </div>
    );
  }

  if (error) {
    return (
      <div
        className={cn("flex min-h-screen w-full items-center justify-center bg-gray-50", appDarkModeClasses.surface)}
      >
        <div className="rounded-md border border-red-300 bg-white px-4 py-3 text-sm text-red-600 dark:border-red-800 dark:bg-gray-900 dark:text-red-400">
          Failed to load workspaces.
        </div>
      </div>
    );
  }

  const lastVisited = account?.id ? readLastVisitedFactory(account.id, organizationId) : null;
  const targetFactory = pickReadyFactory(factories, lastVisited);

  if (targetFactory?.key) {
    return (
      <Navigate to={factoryHomePath(organizationId, targetFactory.key, firstFactoryLineId(targetFactory))} replace />
    );
  }

  if (permissionsLoading) {
    return (
      <div
        className={cn("flex min-h-screen w-full items-center justify-center bg-gray-50", appDarkModeClasses.surface)}
      >
        <p className="text-sm text-gray-500 dark:text-gray-400">Loading workspaces…</p>
      </div>
    );
  }

  if (canAct("factories", "create")) {
    return <Navigate to={newFactoryPath(organizationId)} replace />;
  }

  return (
    <div
      className={cn(
        "flex min-h-screen w-full items-center justify-center bg-background px-6",
        appDarkModeClasses.surface,
      )}
      data-testid="factories-empty-state"
    >
      <div className="max-w-md rounded-lg border border-border bg-card p-8 text-center">
        <h1 className="text-[16px] font-semibold">An organization admin must finish setup</h1>
        <p className="mt-2 text-[13px] text-muted-foreground">
          Ask an organization admin to connect the integrations and configure this workspace.
        </p>
      </div>
    </div>
  );
}
