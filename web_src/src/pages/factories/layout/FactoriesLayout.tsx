import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { Link } from "@/components/Link/link";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateFactory, useFactories, useFactory, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useFactoryWebsocket } from "@/hooks/useFactoryWebsocket";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { AlertTriangle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Outlet, useNavigate, useParams } from "react-router-dom";
import { CreateFactoryDialog } from "../CreateFactoryDialog";
import { factoryDetailPath, factoryListPath } from "../lib/factoryPagePaths";
import { clearLastVisitedFactory, recordLastVisitedFactory } from "../lib/lastVisitedFactory";
import { FactoriesLayoutContext } from "./factoriesLayoutContext";
import { FactoriesNav } from "./FactoriesNav";
import { SidebarUserMenu } from "./SidebarUserMenu";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

const MAX_RECENT_WORK_ORDERS = 5;

export function FactoriesLayout() {
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();

  if (!organizationId || !factoryId) {
    return null;
  }

  return <FactoriesLayoutContent organizationId={organizationId} factoryId={factoryId} />;
}

function FactoriesLayoutContent({ organizationId, factoryId }: { organizationId: string; factoryId: string }) {
  const navigate = useNavigate();
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [createFactoryOpen, setCreateFactoryOpen] = useState(false);

  const { data: factories = [] } = useFactories(organizationId);
  const { data: organization } = useOrganization(organizationId);
  const { data: factory, error: factoryError } = useFactory(organizationId, factoryId);
  useFactoryWebsocket(organizationId, factoryId);
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);

  const createFactory = useCreateFactory(organizationId);

  const pageTitle = useMemo(() => (factory?.name ? [factory.name] : ["Factories"]), [factory?.name]);
  usePageTitle(pageTitle);

  useEffect(() => {
    if (account?.id && factory?.id) {
      recordLastVisitedFactory(account.id, organizationId, factory.id);
    }
  }, [account?.id, organizationId, factory?.id]);

  // A stale last-visited pointer would send us back here in a loop after the
  // error state's "Back to workspaces" link. Clear it as soon as the URL's
  // factoryId is confirmed unreachable.
  useEffect(() => {
    if (account?.id && factoryError) {
      clearLastVisitedFactory(account.id, organizationId, factoryId);
    }
  }, [account?.id, factoryError, factoryId, organizationId]);

  const recentWorkOrders = useMemo(() => sortRecentWorkOrders(workOrders), [workOrders]);

  const layoutContextValue = useMemo(
    () => ({
      organizationId,
      factoryId,
      factory: factory ?? null,
      factories,
    }),
    [organizationId, factoryId, factory, factories],
  );

  const handleCreateFactory = async (input: { name: string; description: string }) => {
    const created = await createFactory.mutateAsync(input);
    setCreateFactoryOpen(false);
    if (created.id) {
      navigate(factoryDetailPath(organizationId, created.id));
    }
  };

  if (factoryError) {
    return <FactoriesLayoutError organizationId={organizationId} />;
  }

  if (!factory) {
    return <FactoriesLayoutLoading />;
  }

  return (
    <FactoriesLayoutContext.Provider value={layoutContextValue}>
      <div
        className={cn("flex min-h-screen w-full bg-gray-50", appDarkModeClasses.surface)}
        data-testid="factories-layout"
      >
        <FactoriesSidebar
          organizationId={organizationId}
          factoryId={factoryId}
          factory={factory}
          factories={factories}
          organizationName={organization?.metadata?.name ?? ""}
          accountName={account?.name}
          accountEmail={account?.email}
          accountAvatarUrl={account?.avatar_url}
          canOpenSettings={canAct("factories", "update")}
          canCreateFactory={canAct("factories", "create")}
          permissionsLoading={permissionsLoading}
          recentWorkOrders={recentWorkOrders}
          onOpenCreateFactory={() => setCreateFactoryOpen(true)}
        />
        <main className="flex min-h-screen min-w-0 flex-1 flex-col">
          <Outlet />
        </main>
      </div>

      <CreateFactoryDialog
        open={createFactoryOpen}
        isSaving={createFactory.isPending}
        onClose={() => setCreateFactoryOpen(false)}
        onCreate={handleCreateFactory}
      />
    </FactoriesLayoutContext.Provider>
  );
}

interface FactoriesSidebarProps {
  organizationId: string;
  factoryId: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
  organizationName: string;
  accountName?: string | null;
  accountEmail?: string | null;
  accountAvatarUrl?: string | null;
  canOpenSettings: boolean;
  canCreateFactory: boolean;
  permissionsLoading: boolean;
  recentWorkOrders: FactoriesWorkOrder[];
  onOpenCreateFactory: () => void;
}

function FactoriesSidebar({
  organizationId,
  factoryId,
  factory,
  factories,
  organizationName,
  accountName,
  accountEmail,
  accountAvatarUrl,
  canOpenSettings,
  canCreateFactory,
  permissionsLoading,
  recentWorkOrders,
  onOpenCreateFactory,
}: FactoriesSidebarProps) {
  return (
    <aside
      className={cn(
        "flex w-60 shrink-0 flex-col border-r border-slate-950/10 bg-white",
        "dark:border-gray-700/70 dark:bg-gray-950",
      )}
      data-testid="factories-sidebar"
    >
      <WorkspaceSwitcher
        organizationId={organizationId}
        factory={factory}
        factories={factories}
        canOpenSettings={canOpenSettings}
        canCreateFactory={canCreateFactory}
        permissionsLoading={permissionsLoading}
        onCreateFactory={onOpenCreateFactory}
      />
      <div className="flex-1 overflow-y-auto">
        <FactoriesNav organizationId={organizationId} factoryId={factoryId} recentWorkOrders={recentWorkOrders} />
      </div>
      <SidebarUserMenu
        organizationId={organizationId}
        userName={accountName ?? "You"}
        userEmail={accountEmail ?? undefined}
        userAvatarUrl={accountAvatarUrl}
        organizationName={organizationName || "Organization"}
      />
    </aside>
  );
}

function FactoriesLayoutLoading() {
  return (
    <div className={cn("flex min-h-screen items-center justify-center bg-gray-50", appDarkModeClasses.surface)}>
      <p className="text-sm text-gray-500 dark:text-gray-400">Loading workspace…</p>
    </div>
  );
}

function FactoriesLayoutError({ organizationId }: { organizationId: string }) {
  return (
    <div
      className={cn("flex min-h-screen items-center justify-center bg-gray-50 px-6", appDarkModeClasses.surface)}
      data-testid="factories-layout-error"
    >
      <div className="max-w-md rounded-lg border border-slate-950/10 bg-white p-8 text-center dark:border-gray-700/70 dark:bg-gray-900">
        <div className="mx-auto mb-4 flex h-10 w-10 items-center justify-center rounded-full bg-red-50 text-red-600 dark:bg-red-500/10 dark:text-red-400">
          <AlertTriangle className="h-5 w-5" aria-hidden />
        </div>
        <h1 className="text-lg font-semibold tracking-tight text-slate-900 dark:text-gray-100">
          This workspace can't be opened
        </h1>
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          It may have been deleted or you may not have access to it.
        </p>
        <Link
          href={factoryListPath(organizationId)}
          className="mt-6 inline-flex items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 dark:bg-gray-100 dark:text-slate-900 dark:hover:bg-white"
        >
          Back to workspaces
        </Link>
      </div>
    </div>
  );
}

// Newest first, based on updatedAt then createdAt.
function sortRecentWorkOrders<T extends { updatedAt?: string; createdAt?: string }>(orders: T[]): T[] {
  return [...orders]
    .sort((a, b) => {
      const aTime = Date.parse(a.updatedAt ?? a.createdAt ?? "") || 0;
      const bTime = Date.parse(b.updatedAt ?? b.createdAt ?? "") || 0;
      return bTime - aTime;
    })
    .slice(0, MAX_RECENT_WORK_ORDERS);
}
