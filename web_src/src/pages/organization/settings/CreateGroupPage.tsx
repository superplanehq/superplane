import { Heading } from "@/components/Heading/heading";
import { usePageTitle } from "@/hooks/usePageTitle";
import React, { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Breadcrumbs } from "../../../components/Breadcrumbs/breadcrumbs";
import {
  Dropdown,
  DropdownButton,
  DropdownDescription,
  DropdownItem,
  DropdownLabel,
  DropdownMenu,
} from "../../../components/Dropdown/dropdown";
import { Icon } from "../../../components/Icon";
import { Input } from "../../../components/Input/input";
import { Text } from "../../../components/Text/text";
import { useCreateGroup, useOrganizationRoles } from "../../../hooks/useOrganizationData";
import { Button } from "../../../ui/button";

export function CreateGroupPage() {
  const navigate = useNavigate();
  const { organizationId } = useParams<{ organizationId: string }>();
  const orgId = organizationId;
  usePageTitle(["Create Group"]);

  const [groupName, setGroupName] = useState("");
  const [groupDescription, setGroupDescription] = useState("");
  const [selectedRole, setSelectedRole] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { data: roles = [], isLoading: loadingRoles } = useOrganizationRoles(orgId || "");
  const createGroupMutation = useCreateGroup(orgId || "");

  useEffect(() => {
    if (roles.length > 0 && !selectedRole) {
      setSelectedRole(roles[0].metadata?.name || "");
    }
  }, [roles, selectedRole]);

  const handleCreateGroup = async () => {
    if (!groupName.trim() || !selectedRole || !orgId) return;

    setIsCreating(true);
    setError(null);

    try {
      await createGroupMutation.mutateAsync({
        organizationId: orgId,
        groupName: groupName.trim().toLocaleLowerCase().replace(/\s+/g, "_"),
        role: selectedRole,
        displayName: groupName,
        description: groupDescription,
      });

      navigate(`/${orgId}/settings/groups`);
    } catch {
      setError("Failed to create group. Please try again.");
    } finally {
      setIsCreating(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleCreateGroup();
    }
  };

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900 text-left">
      <div className="max-w-6xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="mb-4">
            <Breadcrumbs
              items={[
                {
                  label: "Groups",
                  onClick: () => navigate(`/${orgId}/settings/groups`),
                },
                {
                  label: "Create new group",
                  current: true,
                },
              ]}
              showDivider={false}
            />
          </div>

          <div className="text-left">
            <Heading level={2} className="mb-2">
              Create New Group
            </Heading>
            <Text className="text-zinc-600 dark:text-zinc-400">
              Create a group to organize members and assign roles
            </Text>
          </div>
        </div>

        {/* Create Group Form */}
        <div className="space-y-6">
          <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
            {error && (
              <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-6">
                <p className="text-sm">{error}</p>
              </div>
            )}

            <div className="space-y-6">
              {/* Group Name */}
              <div>
                <label className="block text-sm font-medium text-zinc-900 dark:text-white mb-2">Group Name *</label>
                <Input
                  type="text"
                  placeholder="Enter group name"
                  value={groupName}
                  onChange={(e) => setGroupName(e.target.value)}
                  onKeyPress={handleKeyPress}
                  className="w-full max-w-lg"
                />
              </div>

              {/* Group Description */}
              <div>
                <label className="block text-sm font-medium text-zinc-900 dark:text-white mb-2">
                  Group Description
                </label>
                <textarea
                  placeholder="Describe the group's purpose and responsibilities..."
                  value={groupDescription}
                  onChange={(e) => setGroupDescription(e.target.value)}
                  className="w-full max-w-lg px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-zinc-800 dark:text-white resize-none"
                  rows={3}
                />
              </div>

              {/* Role Selection */}
              <div>
                <label className="block text-sm font-medium text-zinc-900 dark:text-white mb-2">Role *</label>
                {loadingRoles ? (
                  <div className="flex justify-center items-center h-12">
                    <p className="text-zinc-500 dark:text-zinc-400">Loading roles...</p>
                  </div>
                ) : roles.length === 0 ? (
                  <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
                    <div className="flex max-w-lg">
                      <Icon name="warning" className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mr-3 mt-0.5" />
                      <div className="text-sm">
                        <p className="text-yellow-800 dark:text-yellow-200 font-medium">No roles available</p>
                        <p className="text-yellow-700 dark:text-yellow-300 mt-1">
                          Create a role first to assign it to this group.
                        </p>
                        <Link
                          to={`/${orgId}/settings/create-role`}
                          className="inline-flex items-center gap-1 mt-2 text-yellow-800 dark:text-yellow-200 hover:text-yellow-900 dark:hover:text-yellow-100 font-medium"
                        >
                          <Icon name="add" size="sm" />
                          Create Role
                        </Link>
                      </div>
                    </div>
                  </div>
                ) : (
                  <Dropdown>
                    <DropdownButton className="flex items-center gap-2 text-sm justify-between">
                      {roles.find((r) => r.metadata?.name === selectedRole)?.spec?.displayName || "Select Role"}
                      <Icon name="keyboard_arrow_down" />
                    </DropdownButton>
                    <DropdownMenu>
                      {roles.map((role) => (
                        <DropdownItem
                          key={role.metadata?.name}
                          onClick={() => setSelectedRole(role.metadata?.name || "")}
                        >
                          <DropdownLabel>{role.spec?.displayName}</DropdownLabel>
                          <DropdownDescription>{role.spec?.description || ""}</DropdownDescription>
                        </DropdownItem>
                      ))}
                    </DropdownMenu>
                  </Dropdown>
                )}
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-3">
            <Button
              onClick={handleCreateGroup}
              disabled={!groupName.trim() || !selectedRole || isCreating || roles.length === 0}
              className="flex items-center gap-2"
            >
              <Icon name="group_add" size="sm" />
              {isCreating ? "Creating..." : "Create Group"}
            </Button>

            <Link to={`/${orgId}/settings/groups`}>
              <Button variant="secondary">Cancel</Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
