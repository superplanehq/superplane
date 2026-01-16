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
import { Applications } from "./Applications";
import { ApplicationDetails } from "./ApplicationDetails";
import SuperplaneLogo from "@/assets/superplane.svg";
import { cn } from "@/lib/utils";
import {
  AppWindow,
  ArrowRightLeft,
  CircleUser,
  Home,
  LogOut,
  Settings,
  Shield,
  User as UserIcon,
  Users,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { isRBACEnabled } from "@/lib/env";

export function OrganizationSettings() {
  const navigate = useNavigate();
  const location = useLocation();
  const { account: user, loading: userLoading } = useAccount();
  const { organizationId } = useParams<{ organizationId: string }>();

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
  };

  const sectionIds = ["profile", "general", "members", "groups", "roles", "applications"];
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
    { id: "canvases", label: "Canvases", href: `/${organizationId}`, Icon: Home },
    { id: "general", label: "Settings", href: `/${organizationId}/settings/general`, Icon: Settings },
    { id: "members", label: "Members", href: `/${organizationId}/settings/members`, Icon: UserIcon },
    { id: "groups", label: "Groups", href: `/${organizationId}/settings/groups`, Icon: Users },
    ...(isRBACEnabled()
      ? [{ id: "roles", label: "Roles", href: `/${organizationId}/settings/roles`, Icon: Shield }]
      : []),
    { id: "applications", label: "Applications", href: `/${organizationId}/settings/applications`, Icon: AppWindow },
    { id: "change-org", label: "Change Organization", href: "/", Icon: ArrowRightLeft },
  ];

  const userLinks: NavLink[] = [
    { id: "profile", label: "Profile", href: `/${organizationId}/settings/profile`, Icon: CircleUser },
    { id: "sign-out", label: "Sign Out", action: () => (window.location.href = "/logout"), Icon: LogOut },
  ];

  const handleLinkClick = (link: NavLink) => {
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
    if (link.id === "applications" && currentSection === "applications") {
      return true;
    }
    return currentSection === link.id;
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
    applications: {
      title: "Applications",
      description: "Connect external tools and services to extend SuperPlane.",
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
      <Sidebar className="w-60 bg-white dark:bg-gray-950 border-r border-gray-300 dark:border-gray-800">
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
                {organizationLinks.map((link) => (
                  <button
                    key={link.id}
                    type="button"
                    onClick={() => handleLinkClick(link)}
                    className={cn(
                      "group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium transition",
                      isLinkActive(link)
                        ? "bg-sky-100 text-gray-800 dark:bg-sky-800/40 dark:text-white"
                        : "text-gray-500 dark:text-gray-300 hover:bg-sky-100 hover:text-gray-900 dark:hover:bg-gray-800",
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
                  </button>
                ))}
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

      <div className="flex-1 overflow-auto bg-gray-100 dark:bg-gray-900">
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
                organization ? (
                  <General organization={organization} />
                ) : (
                  <div className="flex justify-center items-center h-32">
                    <p className="text-gray-500 dark:text-gray-400">Loading...</p>
                  </div>
                )
              }
            />
            <Route path="members" element={<Members organizationId={organizationId || ""} />} />
            <Route path="groups" element={<Groups organizationId={organizationId || ""} />} />
            <Route path="roles" element={<Roles organizationId={organizationId || ""} />} />
            <Route path="integrations" element={<Navigate to="../applications" replace />} />
            <Route path="applications" element={<Applications organizationId={organizationId || ""} />} />
            <Route
              path="applications/:installationId"
              element={<ApplicationDetails organizationId={organizationId || ""} />}
            />
            <Route path="groups/:groupName/members" element={<GroupMembersPage />} />
            <Route path="create-group" element={<CreateGroupPage />} />
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
