import { Routes, Route, Navigate, useNavigate, useLocation } from "react-router-dom";
import { Sidebar, SidebarBody, SidebarSection } from "../../../components/Sidebar/sidebar";
import { General } from "./General";
import { Groups } from "./Groups";
import { Roles } from "./Roles";
import { GroupMembersPage } from "./GroupMembersPage";
import { CreateGroupPage } from "./CreateGroupPage";
import { CreateRolePage } from "./CreateRolePage";
import { Profile } from "./Profile";
import { useOrganization } from "../../../hooks/useOrganizationData";
import { useAccount } from "../../../contexts/AccountContext";
import { useParams } from "react-router-dom";
import { Members } from "./Members";
import { Integrations } from "./Integrations";
import { IntegrationDetails } from "./IntegrationDetails";
import { Secrets } from "./Secrets";
import { SecretDetail } from "./SecretDetail";
import { ServiceAccounts } from "./ServiceAccounts";
import { ServiceAccountDetail } from "./ServiceAccountDetail";
import SuperplaneLogo from "@/assets/superplane.svg";
import { cn } from "@/lib/utils";
import {
  ArrowRightLeft,
  Bot,
  CircleUser,
  Home,
  Key,
  Lock,
  LogOut,
  Plug,
  Settings,
  Shield,
  User as UserIcon,
  Users,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { usePermissions } from "@/contexts/PermissionsContext";
import { PermissionTooltip, RequirePermission } from "@/components/PermissionGate";

export function OrganizationSettings() {
  const navigate = useNavigate();
  const location = useLocation();
  const { account: user, loading: userLoading } = useAccount();
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  // Use React Query hook for organization data
  const { data: organization, isLoading: loading, error } = useOrganization(organizationId || "");

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

  const organizationLinks: NavLink[] = [
    {
      id: "canvases",
      label: "Canvases",
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
      label: "Service Accounts",
      href: `/${organizationId}/settings/service-accounts`,
      Icon: Bot,
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
    { id: "change-org", label: "Change Organization", href: "/", Icon: ArrowRightLeft },
  ];

  const userLinks: NavLink[] = [
    { id: "profile", label: "Profile", href: `/${organizationId}/settings/profile`, Icon: CircleUser },
    { id: "sign-out", label: "Sign Out", action: () => (window.location.href = "/logout"), Icon: LogOut },
  ];

  const handleLinkClick = (link: NavLink) => {
    if (link.permission && !permissionsLoading && !canAct(link.permission.resource, link.permission.action)) {
      return;
    }

    if (link.action) {
      link.action();
      return;
    }

    if (link.href) {
      if (link.href.startsWith("http")) {
        window.location.href = link.href;
      } else {
        navigate(link.href);
      }
    }
  };

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
    secrets: {
      title: "Secrets",
      description: "Store and manage secrets.",
    },
    "service-accounts": {
      title: "Service Accounts",
      description: "Create and manage service accounts for programmatic API access.",
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
    <div className="flex h-screen bg-gray-50 dark:bg-gray-950">
      <Sidebar className="w-60 bg-white dark:bg-gray-800 border-r border-gray-300 dark:border-gray-800">
        <SidebarBody>
          <SidebarSection className="px-4 py-2.5">
            <button
              type="button"
              onClick={() => navigate(`/${organizationId}`)}
              className="w-7 h-7"
              aria-label="Go to Canvases"
            >
              <img src={SuperplaneLogo} alt="SuperPlane" className="w-7 h-7 object-contain" />
            </button>
          </SidebarSection>
          <SidebarSection className="p-4 border-t border-gray-300">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-wide text-gray-100 bg-gray-800 inline px-1 py-0.5 rounded">
                Org
              </p>
              <p className="mt-2 text-sm font-semibold text-gray-800 dark:text-white truncate">{organizationName}</p>
              <div className="mt-3 flex flex-col">
                {organizationLinks.map((link) => {
                  const allowed = canAccessLink(link);
                  const linkButton = (
                    <button
                      key={link.id}
                      type="button"
                      onClick={() => handleLinkClick(link)}
                      disabled={!allowed}
                      className={cn(
                        "group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium transition",
                        isLinkActive(link)
                          ? "bg-sky-100 text-gray-800 dark:bg-sky-800/40 dark:text-white"
                          : "text-gray-500 dark:text-gray-300 hover:bg-sky-100 hover:text-gray-900 dark:hover:bg-gray-800",
                        !allowed && "opacity-60 cursor-not-allowed hover:bg-transparent hover:text-gray-500",
                      )}
                    >
                      <link.Icon
                        size={16}
                        className={cn(
                          "text-gray-500 transition group-hover:text-gray-900 dark:group-hover:text-white",
                          isLinkActive(link) && "text-gray-800 dark:text-white",
                        )}
                      />
                      <span className="truncate">{link.label}</span>
                      {!allowed && <Lock size={12} className="ml-auto text-gray-400" />}
                    </button>
                  );

                  if (allowed) {
                    return linkButton;
                  }

                  return (
                    <PermissionTooltip
                      key={link.id}
                      allowed={false}
                      message={`You don't have permission to view ${link.label.toLowerCase()}.`}
                      className="w-full"
                    >
                      {linkButton}
                    </PermissionTooltip>
                  );
                })}
              </div>
            </div>
          </SidebarSection>

          <SidebarSection className="p-4 border-t border-gray-300">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-wide text-white bg-sky-500 inline px-1 py-0.5 rounded">
                You
              </p>
              <div className="mt-2">
                <p className="text-sm font-semibold text-gray-800 dark:text-white truncate">{userName}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{userEmail}</p>
              </div>
              <div className="mt-3 flex flex-col">
                {userLinks.map((link) => (
                  <button
                    key={link.id}
                    type="button"
                    onClick={() => handleLinkClick(link)}
                    className={cn(
                      "group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium transition",
                      isLinkActive(link)
                        ? "bg-sky-100 text-sky-900 dark:bg-sky-800/40 dark:text-white"
                        : "text-gray-500 dark:text-gray-300 hover:bg-sky-100 hover:text-gray-900 dark:hover:bg-gray-800",
                    )}
                  >
                    <link.Icon
                      size={16}
                      className={cn(
                        "text-gray-500 transition group-hover:text-gray-900 dark:group-hover:text-white",
                        isLinkActive(link) && "text-sky-900 dark:text-white",
                      )}
                    />
                    <span className="truncate">{link.label}</span>
                  </button>
                ))}
              </div>
            </div>
          </SidebarSection>
        </SidebarBody>
      </Sidebar>

      <div className="flex-1 overflow-auto bg-slate-100 dark:bg-slate-900">
        <div className="px-8 pb-8 w-full max-w-3xl mx-auto">
          <div className="pt-10 pb-8">
            <h1 className="!text-2xl font-medium text-gray-900 dark:text-white">{activeMeta.title}</h1>
            <p className="text-sm mt-2 text-gray-800 dark:text-gray-300">{activeMeta.description}</p>
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
              path="integrations/:integrationId"
              element={
                <RequirePermission resource="integrations" action="read">
                  <IntegrationDetails organizationId={organizationId || ""} />
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
            <Route path="secrets" element={<Secrets organizationId={organizationId || ""} />} />
            <Route path="secrets/:secretId" element={<SecretDetail organizationId={organizationId || ""} />} />
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
            <Route path="create-role" element={<CreateRolePage />} />
            <Route path="create-role/:roleName" element={<CreateRolePage />} />
            <Route path="profile" element={<Profile />} />
            <Route
              path="billing"
              element={
                <div className="pt-6">
                  <h1 className="text-2xl font-semibold">Billing & Plans</h1>
                  <p>Billing management coming soon...</p>
                </div>
              }
            />
          </Routes>
        </div>
      </div>
    </div>
  );
}
