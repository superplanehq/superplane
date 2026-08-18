import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { Link } from "@/components/Link/link";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactories, useFactory, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useFactoryWebsocket } from "@/hooks/useFactoryWebsocket";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { AlertTriangle } from "lucide-react";
import { useEffect, useMemo } from "react";
import { Navigate, Outlet, useLocation, useNavigate, useParams } from "react-router";
import { CreateWorkOrderDialog } from "../CreateWorkOrderDialog";
import {
  factoryRouteNeedsCanonicalRedirect,
  replaceFactoryKeySegment,
  resolveFactoryByKey,
} from "../lib/factoryKeyResolution";
import { factoryListPath, newFactoryPath } from "../lib/factoryPagePaths";
import { clearLastVisitedFactory, recordLastVisitedFactory } from "../lib/lastVisitedFactory";
import { useFactoriesThemeClass } from "../lib/useFactoriesThemeClass";
import { useOnboardingStorybook } from "../pages/onboarding/useOnboardingStorybook";
import { isFactoryOnboardingComplete } from "../pages/onboarding/onboardingStatus";
import { FactoriesLayoutContext } from "./factoriesLayoutContext";
import { FactoriesNav } from "./FactoriesNav";
import { SidebarUserMenu } from "./SidebarUserMenu";
import { useCreateWorkOrderDialogState } from "./useCreateWorkOrderDialogState";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

const MAX_RECENT_WORK_ORDERS = 5;

function isOnboardingSidebarHidden(pendingWorkspaceId: string | undefined, factoryId: string) {
  return Boolean(pendingWorkspaceId && pendingWorkspaceId === factoryId);
}

/**
 * Hide workspace chrome while onboarding runs so the setup wizard matches the
 * onboarding design. Storybook uses its pending pointer; production checks the
 * server-backed onboarding state for admins who run setup.
 */
function shouldHideOnboardingSidebar(args: {
  storybookPendingWorkspaceId: string | undefined;
  hasStorybookOnboarding: boolean;
  factoryId: string;
  canConfigure: boolean;
  factory: FactoriesFactory | null;
}): boolean {
  if (isOnboardingSidebarHidden(args.storybookPendingWorkspaceId, args.factoryId)) {
    return true;
  }
  return !args.hasStorybookOnboarding && args.canConfigure && !isFactoryOnboardingComplete(args.factory);
}

export function FactoriesLayout() {
  const { organizationId, factoryKey } = useParams<{ organizationId: string; factoryKey: string }>();

  if (!organizationId || !factoryKey) {
    return null;
  }

  return <FactoriesLayoutResolver organizationId={organizationId} factoryKey={factoryKey} />;
}

/**
 * Resolves the `:factoryKey` route segment to a real factory before handing
 * off to `FactoriesLayoutContent`. Keeps the id/key resolution — and its
 * loading/not-found/redirect states — out of the main layout body.
 */
function FactoriesLayoutResolver({ organizationId, factoryKey }: { organizationId: string; factoryKey: string }) {
  const location = useLocation();
  const {
    data: factories = [],
    isLoading: factoriesLoading,
    isFetching: factoriesFetching,
  } = useFactories(organizationId);
  // `isFetching` (not just `isLoading`) so a just-created workspace — whose
  // list invalidation is still in flight when we navigate to its new key —
  // shows the loading state instead of flashing "workspace not found".
  const resolution = resolveFactoryByKey(factories, factoryKey, factoriesLoading || factoriesFetching);

  if (factoryRouteNeedsCanonicalRedirect(resolution, factoryKey)) {
    const target = replaceFactoryKeySegment(location.pathname, organizationId, factoryKey, resolution.factory!.key!);
    return <Navigate to={`${target}${location.search}`} replace />;
  }

  if (resolution.status === "not-found") {
    return <FactoriesLayoutError organizationId={organizationId} />;
  }

  if (resolution.status === "loading" || !resolution.factory?.id) {
    return <FactoriesLayoutLoading />;
  }

  return (
    <FactoriesLayoutContent
      organizationId={organizationId}
      factoryId={resolution.factory.id}
      factoryKey={resolution.factory.key ?? factoryKey}
      factories={factories}
    />
  );
}

function FactoriesLayoutContent({
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
  useFactoriesThemeClass();
  const navigate = useNavigate();
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { createWorkOrderOpen, openCreateWorkOrder, closeCreateWorkOrder, completeCreateWorkOrder } =
    useCreateWorkOrderDialogState(organizationId, factoryKey, canAct("work_orders", "create"));

  const { data: organization } = useOrganization(organizationId);
  const { data: factory, error: factoryError } = useFactory(organizationId, factoryId);
  useFactoryWebsocket(organizationId, factoryId);
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);

  const storybookOnboarding = useOnboardingStorybook();

  const pageTitle = useMemo(() => (factory?.name ? [factory.name] : ["Workspaces"]), [factory?.name]);
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
      factoryKey,
      factory: factory ?? null,
      factories,
      openCreateWorkOrder,
    }),
    [organizationId, factoryId, factoryKey, factory, factories, openCreateWorkOrder],
  );

  if (factoryError) {
    return <FactoriesLayoutError organizationId={organizationId} />;
  }

  if (!factory) {
    return <FactoriesLayoutLoading />;
  }

  // Incomplete workspaces show only the setup wizard, without workspace
  // chrome, so the production flow matches the onboarding design.
  const hideSidebar = shouldHideOnboardingSidebar({
    storybookPendingWorkspaceId: storybookOnboarding?.pending?.workspaceId,
    hasStorybookOnboarding: Boolean(storybookOnboarding),
    factoryId,
    canConfigure: canAct("factories", "update"),
    factory,
  });

  return (
    <FactoriesLayoutContext.Provider value={layoutContextValue}>
      <div className="flex h-screen w-full bg-background text-foreground" data-testid="factories-layout">
        {hideSidebar ? null : (
          <FactoriesSidebar
            organizationId={organizationId}
            factoryKey={factoryKey}
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
            onOpenCreateFactory={() => navigate(newFactoryPath(organizationId))}
          />
        )}
        <main className="relative min-h-0 min-w-0 flex-1 overflow-y-auto bg-background">
          <Outlet />
        </main>
      </div>

      {canAct("work_orders", "create") ? (
        <CreateWorkOrderDialog
          open={createWorkOrderOpen}
          onClose={closeCreateWorkOrder}
          onCreated={completeCreateWorkOrder}
        />
      ) : null}
    </FactoriesLayoutContext.Provider>
  );
}

interface FactoriesSidebarProps {
  organizationId: string;
  factoryKey: string;
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
  factoryKey,
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
      className="sticky top-0 flex h-screen w-[var(--workspace-navigation-width)] shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
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
        <FactoriesNav organizationId={organizationId} factoryKey={factoryKey} recentWorkOrders={recentWorkOrders} />
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
    <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
      <p className="text-[13px] text-muted-foreground">Loading workspace…</p>
    </div>
  );
}

function FactoriesLayoutError({ organizationId }: { organizationId: string }) {
  return (
    <div
      className="flex min-h-screen items-center justify-center bg-background px-6 text-foreground"
      data-testid="factories-layout-error"
    >
      <div className="max-w-md rounded-lg border border-border bg-card p-8 text-center">
        <div className="mx-auto mb-4 flex h-10 w-10 items-center justify-center rounded-full bg-destructive/10 text-destructive">
          <AlertTriangle className="h-5 w-5" aria-hidden />
        </div>
        <h1 className="text-[16px] font-semibold tracking-[-0.01em] text-foreground">This workspace can't be opened</h1>
        <p className="mt-2 text-[13px] text-muted-foreground">
          It may have been deleted or you may not have access to it.
        </p>
        <Link
          href={factoryListPath(organizationId)}
          className="mt-6 inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-[13px] font-medium text-primary-foreground hover:opacity-90"
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
