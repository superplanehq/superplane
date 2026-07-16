import { Routes, Route, Navigate, Link, useLocation, matchPath } from "react-router-dom";
import { Sidebar, SidebarBody, SidebarSection } from "../../../components/Sidebar/sidebar";
import { General } from "./General";
import { Groups } from "./Groups";
import { Roles } from "./Roles";
import { GroupMembersPage } from "./GroupMembersPage";
import { CreateGroupPage } from "./CreateGroupPage";
import { CreateRolePage } from "./CreateRolePage";
import { Profile } from "./Profile";
import { useOrganization } from "../../../hooks/useOrganizationData";
import { useAccount } from "../../../contexts/useAccount";
import { useParams } from "react-router-dom";
import { Members } from "./Members";
import { Integrations } from "./Integrations";
import { Secrets } from "./Secrets";
import { SecretDetail } from "./SecretDetail";
import { ServiceAccounts } from "./ServiceAccounts";
import { ServiceAccountDetail } from "./ServiceAccountDetail";
import { Usage } from "./Usage";
import SuperplaneLogo from "@/assets/superplane.svg";
import { isUsagePageForced } from "@/lib/env";
import { cn } from "@/lib/utils";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import {
  ArrowRightLeft,
  Gauge,
  CircleUser,
  Home,
  Key,
  KeyRound,
  Lock,
  LogOut,
  Plug,
  Settings,
  Shield,
  User as UserIcon,
  Users,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { usePermissions } from "@/contexts/usePermissions";
import { PermissionTooltip, RequireAnyPermission, RequirePermission } from "@/components/PermissionGate";
import { useOrganizationUsage } from "@/hooks/useOrganizationData";
import { IntegrationDetailsRoute } from "./components/IntegrationDetailsRoute";
import { IntegrationSetup } from "./components/IntegrationSetup";
import { ThemePreferenceControl } from "@/components/ThemePreferenceControl";

function settingsSidebarNavLinkClass(active: boolean) {
  return cn(
    "group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium transition-colors",
    active
      ? "bg-sky-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100"
      : "text-gray-500 hover:bg-sky-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100",
  );
}

function settingsSidebarNavIconClass(active: boolean) {
  return cn(
    "text-gray-500 transition-colors group-hover:text-gray-800 dark:text-gray-400 dark:group-hover:text-gray-100",
    active && "text-gray-800 dark:text-gray-100",
  );
}

export function OrganizationSettings() {
  const location = useLocation();
  const { account: user, loading: userLoading } = useAccount();
  const { organizationId } = useParams<{ organizationId: string }>();
  const isIntegrationSetupRoute = Boolean(
    matchPath({ path: "/:organizationId/settings/integrations/:integrationName/setup", end: true }, location.pathname),
  );
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canReadOrg = permissionsLoading || canAct("org", "read");

  // Use React Query hook for organization data
  const { data: organization, isLoading: loading, error } = useOrganization(organizationId || "");
  const { data: usageStatus, error: usageError } = useOrganizationUsage(
    organizationId || "",
    !!organizationId && canReadOrg,
  );

  if (userLoading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-gray-500 dark:text-gray-400">Loading user...</p>
      </div>
    );
  }

  if (!organizationId) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-gray-500 dark:text-gray-400">Organization not found</p>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-gray-500 dark:text-gray-400">Loading organization...</p>
      </div>
    );
  }

  if (error || (!loading && !organization)) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-gray-500 dark:text-gray-400">
          {error instanceof Error ? error.message : "Organization not found"}
        </p>
      </div>
    );
  }

  type NavLink = {
    id: string;
    label: string;
    href?: string;
    action?: () => void;
    Icon: LucideIcon;
    permission?: { resource: string; action: string };
  };

  const sectionIds = [
    "profile",
    "general",
    "members",
    "groups",
    "roles",
    "integrations",
    "secrets",
    "service-accounts",
    "billing",
  ];
  const pathSegments = location.pathname?.split("/").filter(Boolean) || [];
  const settingsIndex = pathSegments.indexOf("settings");
  const segmentsAfterSettings = settingsIndex >= 0 ? pathSegments.slice(settingsIndex + 1) : [];
  const currentSection = segmentsAfterSettings.includes("create-role")
    ? "roles"
    : segmentsAfterSettings.includes("create-group")
      ? "groups"
      : segmentsAfterSettings.find((segment) => sectionIds.includes(segment)) ||
        (sectionIds.includes(pathSegments[pathSegments.length - 1])
          ? pathSegments[pathSegments.length - 1]
          : "general");

  const organizationName = organization?.metadata?.name || "Organization";
  const userName = user?.name || "My Account";
  const userEmail = user?.email || "";
  const usageEnabled =
    usageStatus?.enabled === true || !!usageError || currentSection === "billing" || isUsagePageForced();

  const organizationLinks: NavLink[] = [
    {
      id: "canvases",
      label: "Apps",
      href: `/${organizationId}`,
      Icon: Home,
      permission: { resource: "canvases", action: "read" },
    },
    {
      id: "general",
      label: "Settings",
      href: `/${organizationId}/settings/general`,
      Icon: Settings,
      permission: { resource: "org", action: "read" },
    },
    {
      id: "members",
      label: "Members",
      href: `/${organizationId}/settings/members`,
      Icon: UserIcon,
      permission: { resource: "members", action: "read" },
    },
    {
      id: "service-accounts",
      label: "API Keys",
      href: `/${organizationId}/settings/service-accounts`,
      Icon: KeyRound,
      permission: { resource: "service_accounts", action: "read" },
    },
    {
      id: "groups",
      label: "Groups",
      href: `/${organizationId}/settings/groups`,
      Icon: Users,
      permission: { resource: "groups", action: "read" },
    },
    {
      id: "roles",
      label: "Roles",
      href: `/${organizationId}/settings/roles`,
      Icon: Shield,
      permission: { resource: "roles", action: "read" },
    },
    {
      id: "integrations",
      label: "Integrations",
      href: `/${organizationId}/settings/integrations`,
      Icon: Plug,
      permission: { resource: "integrations", action: "read" },
    },
    {
      id: "secrets",
      label: "Secrets",
      href: `/${organizationId}/settings/secrets`,
      Icon: Key,
      permission: { resource: "secrets", action: "read" },
    },
    { id: "change-org", label: "Change Organization", href: "/?select=true", Icon: ArrowRightLeft },
  ];

  if (usageEnabled) {
    organizationLinks.splice(6, 0, {
      id: "billing",
      label: "Usage",
      href: `/${organizationId}/settings/billing`,
      Icon: Gauge,
      permission: { resource: "org", action: "read" },
    });
  }

  const userLinks: NavLink[] = [
    { id: "profile", label: "Profile", href: `/${organizationId}/settings/profile`, Icon: CircleUser },
    { id: "sign-out", label: "Sign Out", action: () => (window.location.href = "/logout"), Icon: LogOut },
  ];

  const isLinkActive = (link: NavLink) => {
    if (link.id === "canvases") {
      return location.pathname === `/${organizationId}`;
    }
    if (link.id === "change-org" || link.id === "sign-out") {
      return false;
    }
    if (link.id === "integrations" && currentSection === "integrations") {
      return true;
    }
    if (link.id === "secrets" && currentSection === "secrets") {
      return true;
    }
    if (link.id === "service-accounts" && currentSection === "service-accounts") {
      return true;
    }
    return currentSection === link.id;
  };

  const canAccessLink = (link: NavLink) => {
    if (!link.permission) return true;
    if (permissionsLoading) return true;
    return canAct(link.permission.resource, link.permission.action);
  };

  const sectionMeta: Record<
    string,
    {
      title: string;
      description: string;
    }
  > = {
    general: {
      title: "Settings",
      description: "Manage your organization basics.",
    },
    members: {
      title: "Members",
      description: "Invite people and manage who has access to this organization.",
    },
    groups: {
      title: "Groups",
      description: "Organize members into groups to simplify permissions and collaboration.",
    },
    roles: {
      title: "Roles",
      description: "Define fine-grained access by creating and assigning roles.",
    },
    integrations: {
      title: "Integrations",
      description: "Connect external tools and services to extend SuperPlane.",
    },
    billing: {
      title: "Usage",
      description: "Review organization limits and tracked usage for this organization.",
    },
    secrets: {
      title: "Secrets",
      description: "Store and manage secrets.",
    },
    "service-accounts": {
      title: "API Keys",
      description: "Create and manage API keys for programmatic access.",
    },
    profile: {
      title: "Profile",
      description: "Update your personal account information and preferences.",
    },
  };

  const activeMeta = sectionMeta[currentSection] || {
    title: "Organization",
    description: "Manage your organization configuration and resources.",
  };

  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      <Sidebar className={cn("w-60 border-r bg-white", appDarkModeClasses.sidebarEdge, appDarkModeClasses.surface)}>
        <SidebarBody>
          <SidebarSection className="px-4 py-2.5">
            <Link to={`/${organizationId}`} className="block h-7 w-7" aria-label="Go to Apps">
              <img
                src={SuperplaneLogo}
                alt="SuperPlane"
                className="h-7 w-7 object-contain dark:brightness-0 dark:invert"
              />
            </Link>
          </SidebarSection>
          <SidebarSection className={cn("border-t p-4", appDarkModeClasses.sidebarDivider)}>
            <div>
              <p className="inline rounded bg-gray-800 px-1 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-gray-100 dark:bg-gray-300 dark:text-gray-950">
                Org
              </p>
              <p className="mt-2 truncate text-sm font-semibold text-gray-800 dark:text-gray-100">{organizationName}</p>
              <div className="mt-3 flex flex-col">
                {organizationLinks.map((link) => {
                  const allowed = canAccessLink(link);

                  if (!allowed) {
                    return (
                      <PermissionTooltip
                        key={link.id}
                        allowed={false}
                        message={`You don't have permission to view ${link.label.toLowerCase()}.`}
                        className="w-full"
                      >
                        <button
                          type="button"
                          disabled
                          className={cn(
                            "group flex w-full items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium transition-colors",
                            "cursor-not-allowed text-gray-500 opacity-60 hover:bg-transparent hover:text-gray-500 dark:text-gray-400 dark:hover:bg-transparent dark:hover:text-gray-400",
                          )}
                        >
                          <link.Icon size={16} className="text-gray-500 dark:text-gray-400" />
                          <span className="truncate">{link.label}</span>
                          <Lock size={12} className="ml-auto text-gray-400" />
                        </button>
                      </PermissionTooltip>
                    );
                  }

                  if (link.href) {
                    return (
                      <Link key={link.id} to={link.href} className={settingsSidebarNavLinkClass(isLinkActive(link))}>
                        <link.Icon size={16} className={settingsSidebarNavIconClass(isLinkActive(link))} />
                        <span className="truncate">{link.label}</span>
                      </Link>
                    );
                  }

                  return (
                    <button
                      key={link.id}
                      type="button"
                      onClick={link.action}
                      className={settingsSidebarNavLinkClass(isLinkActive(link))}
                    >
                      <link.Icon size={16} className={settingsSidebarNavIconClass(isLinkActive(link))} />
                      <span className="truncate">{link.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          </SidebarSection>

          <SidebarSection className={cn("border-t p-4", appDarkModeClasses.sidebarDivider)}>
            <div>
              <p className="inline rounded bg-sky-500 px-1 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-white dark:bg-sky-300 dark:text-sky-950">
                You
              </p>
              <div className="mt-2">
                <p className="truncate text-sm font-semibold text-gray-800 dark:text-gray-100">{userName}</p>
                <p className="truncate text-[13px] font-medium text-gray-500 dark:text-gray-400">{userEmail}</p>
              </div>
              <div className="mt-3 flex flex-col">
                {userLinks.map((link) =>
                  link.href ? (
                    <Link key={link.id} to={link.href} className={settingsSidebarNavLinkClass(isLinkActive(link))}>
                      <link.Icon size={16} className={settingsSidebarNavIconClass(isLinkActive(link))} />
                      <span className="truncate">{link.label}</span>
                    </Link>
                  ) : (
                    <button
                      key={link.id}
                      type="button"
                      onClick={link.action}
                      className={settingsSidebarNavLinkClass(isLinkActive(link))}
                    >
                      <link.Icon size={16} className={settingsSidebarNavIconClass(isLinkActive(link))} />
                      <span className="truncate">{link.label}</span>
                    </button>
                  ),
                )}
                <ThemePreferenceControl />
              </div>
            </div>
          </SidebarSection>
        </SidebarBody>
      </Sidebar>

      <div className={cn("flex-1 overflow-auto bg-slate-100 [scrollbar-gutter:stable]", appDarkModeClasses.surface)}>
        <div className={cn("mx-auto w-full px-8 pb-8", isIntegrationSetupRoute ? "max-w-6xl" : "max-w-3xl")}>
          <div className="pt-10 pb-8">
            <h1 className={cn("!text-2xl font-medium text-gray-900", appDarkModeClasses.textPrimary)}>
              {activeMeta.title}
            </h1>
            <p className={cn("mt-2 text-sm text-gray-800", appDarkModeClasses.textSecondary)}>
              {activeMeta.description}
            </p>
          </div>
          <Routes>
            <Route path="" element={<Navigate to="general" replace />} />
            <Route
              path="general"
              element={
                <RequirePermission resource="org" action="read">
                  {organization ? (
                    <General organization={organization} />
                  ) : (
                    <div className="flex justify-center items-center h-32">
                      <p className="text-gray-500 dark:text-gray-400">Loading...</p>
                    </div>
                  )}
                </RequirePermission>
              }
            />
            <Route
              path="members"
              element={
                <RequirePermission resource="members" action="read">
                  <Members organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="groups"
              element={
                <RequirePermission resource="groups" action="read">
                  <Groups organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="roles"
              element={
                <RequirePermission resource="roles" action="read">
                  <Roles organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="integrations"
              element={
                <RequirePermission resource="integrations" action="read">
                  <Integrations organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="integrations/:integrationName/setup"
              element={
                <RequireAnyPermission
                  checks={[
                    { resource: "integrations", action: "create" },
                    { resource: "integrations", action: "update" },
                  ]}
                >
                  <IntegrationSetup organizationId={organizationId || ""} />
                </RequireAnyPermission>
              }
            />
            <Route
              path="integrations/:integrationId"
              element={
                <RequirePermission resource="integrations" action="read">
                  <IntegrationDetailsRoute organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="secrets"
              element={
                <RequirePermission resource="secrets" action="read">
                  <Secrets organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="groups/:groupName/members"
              element={
                <RequirePermission resource="groups" action="read">
                  <GroupMembersPage />
                </RequirePermission>
              }
            />
            <Route
              path="create-group"
              element={
                <RequirePermission resource="groups" action="create">
                  <CreateGroupPage />
                </RequirePermission>
              }
            />
            <Route
              path="secrets/:secretId"
              element={
                <RequirePermission resource="secrets" action="read">
                  <SecretDetail organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="service-accounts"
              element={
                <RequirePermission resource="service_accounts" action="read">
                  <ServiceAccounts organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="service-accounts/:id"
              element={
                <RequirePermission resource="service_accounts" action="read">
                  <ServiceAccountDetail organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
            <Route
              path="create-role"
              element={
                <RequirePermission resource="roles" action="create">
                  <CreateRolePage />
                </RequirePermission>
              }
            />
            <Route
              path="create-role/:roleName"
              element={
                <RequirePermission resource="roles" action="read">
                  <CreateRolePage />
                </RequirePermission>
              }
            />
            <Route path="profile" element={<Profile />} />
            <Route
              path="billing"
              element={
                <RequirePermission resource="org" action="read">
                  <Usage organizationId={organizationId || ""} />
                </RequirePermission>
              }
            />
          </Routes>
        </div>
      </div>
    </div>
  );
}
