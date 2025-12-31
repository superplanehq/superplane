import React from "react";
import { Avatar } from "./Avatar/avatar";
import {
  Dropdown,
  DropdownButton,
  DropdownDivider,
  DropdownHeader,
  DropdownItem,
  DropdownLabel,
  DropdownMenu,
  DropdownSection,
} from "./Dropdown/dropdown";
import { Text } from "./Text/text";
import { Icon } from "./Icon";
import { Link } from "./Link/link";
import { Button } from "./ui/button";
import { useOrganization } from "../hooks/useOrganizationData";
import { useAccount } from "../contexts/AccountContext";
import { useParams } from "react-router-dom";
import SuperplaneLogo from "@/assets/superplane.svg";
import { isRBACEnabled } from "@/lib/env";

const Navigation: React.FC = () => {
  const { account } = useAccount();
  const { organizationId } = useParams<{ organizationId: string }>();
  const { data: organization } = useOrganization(organizationId || "");
  return (
    <div className="fixed top-0 left-0 right-0 z-50 bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-800 border-b">
      <div className="flex items-center justify-between h-12 px-6">
        <Link href={`/${organizationId}`} className="flex items-center flex-shrink-0 text-decoration-none">
          <img src={SuperplaneLogo} alt="SuperPlane" className="w-8 h-8" />
        </Link>
        <div className="flex items-center flex-shrink-0">
          {/* Merged Account Dropdown */}
          <Dropdown>
            <DropdownButton as={Button} size="sm" variant="outline">
              <span className="text-sm font-medium truncate max-w-[150px]">
                {organization?.metadata?.name || account?.name}
              </span>
              <Avatar
                src={account?.avatar_url}
                initials={
                  account?.name
                    ? account.name
                        .split(" ")
                        .map((n) => n[0])
                        .join("")
                        .toUpperCase()
                    : "?"
                }
                alt={account?.name || "User"}
                className="w-5 h-5 -mr-1"
              />
            </DropdownButton>

            <DropdownMenu className="w-64">
              {/* User Section */}
              <DropdownHeader>
                <div className="flex items-center space-x-3">
                  <Avatar
                    src={account?.avatar_url}
                    initials={
                      account?.name
                        ? account.name
                            .split(" ")
                            .map((n) => n[0])
                            .join("")
                            .toUpperCase()
                        : "?"
                    }
                    alt={account?.name || "User"}
                    className="size-8"
                  />
                  <div className="flex-1 min-w-0">
                    <Text className="font-medium truncate">{account?.name || "Loading..."}</Text>
                    <Text className="text-sm text-gray-500 truncate">{account?.email || "Loading..."}</Text>
                  </div>
                </div>
              </DropdownHeader>

              {/* User Actions */}
              <DropdownSection>
                <DropdownItem href={`/${organizationId}/settings/profile`}>
                  <span className="flex items-center gap-x-2">
                    <Icon name="user-circle" data-slot="icon" size="sm" />
                    <DropdownLabel>Profile</DropdownLabel>
                  </span>
                </DropdownItem>
              </DropdownSection>

              {/* Organization Section */}
              <>
                <DropdownDivider />

                <DropdownHeader>
                  <div className="flex items-center space-x-3">
                    <Avatar
                      initials={(organization?.metadata?.name || "Organization").charAt(0).toUpperCase()}
                      alt={organization?.metadata?.name || "Organization"}
                      className="size-8"
                    />
                    <div className="flex-1 min-w-0">
                      <Text className="font-medium truncate max-w-[150px] truncate-ellipsis">
                        {organization?.metadata?.name || "Organization"}
                      </Text>
                    </div>
                  </div>
                </DropdownHeader>

                {/* Organization Actions */}
                <DropdownSection>
                  <DropdownItem href={`/${organizationId}/settings/general`}>
                    <span className="flex items-center gap-x-2">
                      <Icon name="building-2" data-slot="icon" size="sm" />
                      <DropdownLabel>Organization Settings</DropdownLabel>
                    </span>
                  </DropdownItem>

                  <DropdownItem href={`/${organizationId}/settings/members`}>
                    <span className="flex items-center gap-x-2">
                      <Icon name="user" data-slot="icon" size="sm" />
                      <DropdownLabel>Members</DropdownLabel>
                    </span>
                  </DropdownItem>

                  <DropdownItem href={`/${organizationId}/settings/groups`}>
                    <span className="flex items-center gap-x-2">
                      <Icon name="users" data-slot="icon" size="sm" />
                      <DropdownLabel>Groups</DropdownLabel>
                    </span>
                  </DropdownItem>
                  {isRBACEnabled() && (
                    <DropdownItem href={`/${organizationId}/settings/roles`}>
                      <span className="flex items-center gap-x-2">
                        <Icon name="shield" data-slot="icon" size="sm" />
                        <DropdownLabel>Roles</DropdownLabel>
                      </span>
                    </DropdownItem>
                  )}

                  <DropdownItem href={`/${organizationId}/settings/integrations`}>
                    <span className="flex items-center gap-x-2">
                      <Icon name="plug" data-slot="icon" size="sm" />
                      <DropdownLabel>Integrations</DropdownLabel>
                    </span>
                  </DropdownItem>

                  <DropdownItem href="/">
                    <span className="flex items-center gap-x-2">
                      <Icon name="arrow-left-right" data-slot="icon" size="sm" />
                      <DropdownLabel>Change organization</DropdownLabel>
                    </span>
                  </DropdownItem>
                </DropdownSection>
              </>

              <DropdownDivider />

              {/* Sign Out Section */}
              <DropdownSection>
                <DropdownItem
                  onClick={() => {
                    // Redirect to logout
                    window.location.href = "/logout";
                  }}
                >
                  <span className="flex items-center gap-x-2">
                    <Icon name="log-out" data-slot="icon" size="sm" />
                    <DropdownLabel>Sign Out</DropdownLabel>
                  </span>
                </DropdownItem>
              </DropdownSection>
            </DropdownMenu>
          </Dropdown>
        </div>
      </div>
    </div>
  );
};

export default Navigation;
