import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactories } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { Factory as FactoryIcon, Plus } from "lucide-react";
import { Navigate, useNavigate, useParams } from "react-router";
import { factoryHomePath, firstFactoryLineId, newFactoryPath } from "./lib/factoryPagePaths";
import { pickInitialFactory, readLastVisitedFactory } from "./lib/lastVisitedFactory";
import { useFactoriesThemeClass } from "./lib/useFactoriesThemeClass";

export function FactoriesIndexPage() {
  const { organizationId } = useParams<{ organizationId: string }>();

  if (!organizationId) {
    return null;
  }

  return <FactoriesIndexPageContent organizationId={organizationId} />;
}

function FactoriesIndexPageContent({ organizationId }: { organizationId: string }) {
  const navigate = useNavigate();
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
  const targetFactory = pickInitialFactory(factories, lastVisited);

  if (targetFactory?.key) {
    return (
      <Navigate to={factoryHomePath(organizationId, targetFactory.key, firstFactoryLineId(targetFactory))} replace />
    );
  }

  const canCreate = canAct("factories", "create");

  return (
    <div
      className={cn("flex min-h-screen w-full items-center justify-center bg-gray-50 p-6", appDarkModeClasses.surface)}
      data-testid="factories-empty-state"
    >
      <div className="flex max-w-md flex-col items-center rounded-lg border border-dashed border-slate-300 bg-white px-8 py-12 text-center dark:border-gray-700 dark:bg-gray-900">
        <FactoryIcon className="h-10 w-10 text-slate-400 dark:text-gray-500" aria-hidden />
        <Heading level={2} className="mt-4 text-lg text-slate-900 dark:text-gray-100">
          Create your first workspace
        </Heading>
        <Text className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          Workspaces group tasks, automations, and apps.
        </Text>
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create workspaces."
        >
          <Button
            type="button"
            className="mt-6"
            onClick={() => navigate(newFactoryPath(organizationId))}
            disabled={!canCreate}
            data-testid="factories-index-create-button"
          >
            <Plus className="h-4 w-4" aria-hidden />
            Create workspace
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}
