import { useAccount } from "@/contexts/useAccount";
import { useFactory } from "@/hooks/useFactoryData";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import { Navigate, NavLink, Outlet, useParams } from "react-router-dom";
import { factoryDetailPath, factoryListPath, factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { SidebarUserMenu } from "../../layout/SidebarUserMenu";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FACTORY_SETTINGS_NAV_ITEMS } from "./settingsNavItems";

export function FactorySettingsLayout() {
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();

  if (!organizationId || !factoryId) {
    return null;
  }

  return <FactorySettingsLayoutContent organizationId={organizationId} factoryId={factoryId} />;
}

function FactorySettingsLayoutContent({ organizationId, factoryId }: { organizationId: string; factoryId: string }) {
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId);
  const { data: factory, isLoading, error } = useFactory(organizationId, factoryId);

  usePageTitle(factory?.name ? [factory.name, "Settings"] : ["Settings"]);

  if (!isLoading && error) {
    return <Navigate to={factoryListPath(organizationId)} replace />;
  }

  if (!factory) {
    return (
      <div className={cn("flex min-h-screen items-center justify-center bg-gray-50", appDarkModeClasses.surface)}>
        <p className="text-sm text-gray-500 dark:text-gray-400">Loading settings…</p>
      </div>
    );
  }

  const workspaceGroup = FACTORY_SETTINGS_NAV_ITEMS.filter((item) => item.group === "workspace");
  const governanceGroup = FACTORY_SETTINGS_NAV_ITEMS.filter((item) => item.group === "governance");

  return (
    <FactorySettingsLayoutContext.Provider value={{ organizationId, factoryId, factory }}>
      <div
        className={cn("flex min-h-screen w-full bg-gray-50", appDarkModeClasses.surface)}
        data-testid="factory-settings-layout"
      >
        <aside
          className={cn(
            "flex w-60 shrink-0 flex-col border-r border-slate-950/10 bg-white",
            "dark:border-gray-700/70 dark:bg-gray-950",
          )}
          data-testid="factory-settings-sidebar"
        >
          <NavLink
            to={factoryDetailPath(organizationId, factoryId)}
            className="flex items-center gap-2 border-b border-slate-950/10 px-4 py-3 text-sm text-gray-600 hover:text-gray-900 dark:border-gray-700/70 dark:text-gray-300 dark:hover:text-gray-100"
            data-testid="factory-settings-back"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            Back to workspace
          </NavLink>
          <nav className="flex flex-1 flex-col gap-4 px-2 py-4">
            <SettingsNavGroup organizationId={organizationId} factoryId={factoryId} items={workspaceGroup} />
            <SettingsNavGroup organizationId={organizationId} factoryId={factoryId} items={governanceGroup} />
          </nav>
          <SidebarUserMenu
            organizationId={organizationId}
            userName={account?.name ?? "You"}
            userEmail={account?.email}
            userAvatarUrl={account?.avatar_url}
            organizationName={organization?.metadata?.name ?? "Organization"}
          />
        </aside>
        <main className="flex min-h-screen min-w-0 flex-1 flex-col">
          <Outlet />
        </main>
      </div>
    </FactorySettingsLayoutContext.Provider>
  );
}

function SettingsNavGroup({
  organizationId,
  factoryId,
  items,
}: {
  organizationId: string;
  factoryId: string;
  items: typeof FACTORY_SETTINGS_NAV_ITEMS;
}) {
  return (
    <ul className="flex flex-col gap-0.5">
      {items.map((item) => {
        const Icon = item.Icon;
        return (
          <li key={item.id}>
            <NavLink
              to={factorySettingsSectionPath(organizationId, factoryId, item.id)}
              className={({ isActive }) =>
                cn(
                  "group flex items-center gap-2 rounded-md px-2 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-800 dark:hover:text-gray-100",
                  isActive && "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100",
                )
              }
              data-testid={`factory-settings-nav-${item.id}`}
            >
              <Icon className="h-4 w-4 shrink-0" aria-hidden />
              <span>{item.label}</span>
            </NavLink>
          </li>
        );
      })}
    </ul>
  );
}
