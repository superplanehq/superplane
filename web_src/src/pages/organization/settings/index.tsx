import { Routes, Route, Navigate, useNavigate, useLocation } from "react-router-dom";
import { Avatar } from "../../../components/Avatar/avatar";
import {
  Sidebar,
  SidebarBody,
  SidebarDivider,
  SidebarItem,
  SidebarLabel,
  SidebarSection,
} from "../../../components/Sidebar/sidebar";
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
import { Applications } from "./Applications";
import { ApplicationDetails } from "./ApplicationDetails";

export function OrganizationSettings() {
  const navigate = useNavigate();
  const location = useLocation();
  const { account: user, loading: userLoading } = useAccount();
  const { organizationId } = useParams<{ organizationId: string }>();

  // Use React Query hook for organization data
  const { data: organization, isLoading: loading, error } = useOrganization(organizationId || "");

  // Extract current section from the URL
  const currentSection = location.pathname.split("/").pop() || "general";

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

  const tabs = [
    { id: "profile", label: "Profile", icon: "person" },
    { id: "general", label: "General", icon: "settings" },
    { id: "members", label: "Members", icon: "group" },
    { id: "groups", label: "Groups", icon: "group" },
    { id: "roles", label: "Roles", icon: "admin_panel_settings" },
    { id: "integrations", label: "Integrations", icon: "integration_instructions" },
    { id: "applications", label: "Applications", icon: "apps" },
  ];

  return (
    <div
      className="flex flex-col bg-gray-50 dark:bg-gray-950"
      style={{ height: "calc(100vh - 3rem)", marginTop: "3rem" }}
    >
      <div className="flex flex-1 overflow-hidden">
        <Sidebar className="w-70 bg-white dark:bg-gray-950 border-r bw-1 border-gray-200 dark:border-gray-800">
          <SidebarBody>
            <SidebarSection>
              <div className="flex items-center gap-3 text-sm font-bold py-3">
                <Avatar
                  className="w-6 h-6"
                  src={user?.avatar_url}
                  initials={
                    user?.name
                      ? user.name
                          .split(" ")
                          .map((n) => n[0])
                          .join("")
                          .toUpperCase()
                      : "U"
                  }
                  alt={user?.name || "My Account"}
                />
                <SidebarLabel className="text-gray-900 dark:text-white">{user?.name || "My Account"}</SidebarLabel>
              </div>
              <SidebarItem
                className={`${currentSection === "profile" ? "bg-gray-100 dark:bg-gray-800 rounded-md" : ""}`}
                onClick={() => navigate(`/${organizationId}/settings/profile`)}
              >
                <span className="px-7">
                  <SidebarLabel>Profile</SidebarLabel>
                </span>
              </SidebarItem>
            </SidebarSection>
            <SidebarDivider className="dark:border-gray-800" />
            <SidebarSection>
              <div className="flex items-center gap-3 text-sm font-bold py-3">
                <Avatar
                  className="w-6 h-6 bg-blue-200 dark:bg-blue-800 text-blue-800 dark:text-white"
                  slot="icon"
                  initials={(organization?.metadata?.name || organization?.metadata?.name || "Organization")
                    .charAt(0)
                    .toUpperCase()}
                  alt={organization?.metadata?.name || organization?.metadata?.name || "Organization"}
                />
                <SidebarLabel className="text-gray-900 dark:text-white">
                  {organization?.metadata?.name || organization?.metadata?.name || "Organization"}
                </SidebarLabel>
              </div>
              {tabs
                .filter((tab) => tab.id !== "profile")
                .map((tab) => (
                  <SidebarItem
                    key={tab.id}
                    onClick={() => navigate(`/${organizationId}/settings/${tab.id}`)}
                    className={`${currentSection === tab.id ? "bg-gray-100 dark:bg-gray-800 rounded-md" : ""}`}
                  >
                    <span className={`px-7 ${currentSection === tab.id ? "font-semibold" : "font-normal"}`}>
                      <SidebarLabel>{tab.label}</SidebarLabel>
                    </span>
                  </SidebarItem>
                ))}
            </SidebarSection>
          </SidebarBody>
        </Sidebar>

        <div className="flex-1 overflow-auto bg-gray-50 dark:bg-gray-900">
          <div className="px-8 pb-8">
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
              <Route path="integrations" element={<Integrations organizationId={organizationId || ""} />} />
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
    </div>
  );
}
