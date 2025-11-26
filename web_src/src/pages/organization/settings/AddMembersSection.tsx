import { useState, forwardRef, useImperativeHandle, useMemo } from "react";
import { Button } from "../../../components/Button/button";
import { Input, InputGroup } from "../../../components/Input/input";
import { Avatar } from "../../../components/Avatar/avatar";
import { Checkbox } from "../../../components/Checkbox/checkbox";
import { Icon } from "../../../components/Icon";
import { Text } from "../../../components/Text/text";
import { useOrganizationUsers, useOrganizationGroupUsers, useAddUserToGroup } from "../../../hooks/useOrganizationData";

interface AddMembersSectionProps {
  showRoleSelection?: boolean;
  organizationId: string;
  groupName?: string;
  onMemberAdded?: () => void;
  className?: string;
}

export interface AddMembersSectionRef {
  refreshExistingMembers: () => void;
}

const AddMembersSectionComponent = forwardRef<AddMembersSectionRef, AddMembersSectionProps>(
  ({ organizationId, groupName, onMemberAdded, className }, ref) => {
    const [selectedMembers, setSelectedMembers] = useState<Set<string>>(new Set());
    const [memberSearchTerm, setMemberSearchTerm] = useState("");

    // React Query hooks
    const {
      data: orgUsers = [],
      isLoading: loadingOrgUsers,
      error: orgUsersError,
    } = useOrganizationUsers(organizationId);
    const {
      data: groupUsers = [],
      isLoading: loadingGroupUsers,
      error: groupUsersError,
    } = useOrganizationGroupUsers(organizationId, groupName || "");

    // Mutations
    const addUserToGroupMutation = useAddUserToGroup(organizationId);

    const isInviting = addUserToGroupMutation.isPending;
    const error = orgUsersError || groupUsersError;

    // Calculate available members (org users who aren't in the group)
    const existingMembers = useMemo(() => {
      if (!groupName) return [];

      const existingMemberIds = new Set(groupUsers.map((user) => user.metadata?.id));
      return orgUsers.filter((user) => !existingMemberIds.has(user.metadata?.id));
    }, [orgUsers, groupUsers, groupName]);

    const loadingMembers = loadingOrgUsers || loadingGroupUsers;

    // Expose refresh function to parent
    useImperativeHandle(
      ref,
      () => ({
        refreshExistingMembers: () => {
          // No need to manually refresh - React Query will handle it
        },
      }),
      [],
    );

    const handleExistingMembersSubmit = async () => {
      if (selectedMembers.size === 0) return;

      try {
        const selectedUsers = existingMembers.filter((member) => selectedMembers.has(member.metadata?.id || ""));

        // Process each selected member
        for (const member of selectedUsers) {
          if (groupName) {
            // Add user to specific group - try both userId and email
            try {
              await addUserToGroupMutation.mutateAsync({
                groupName,
                userId: member.metadata?.id || "",
                organizationId,
              });
            } catch (err) {
              // If userId fails, try with email
              if (member.metadata?.email) {
                await addUserToGroupMutation.mutateAsync({
                  groupName,
                  userEmail: member.metadata?.email,
                  organizationId,
                });
              } else {
                throw err;
              }
            }
          }
        }

        setSelectedMembers(new Set());
        setMemberSearchTerm("");

        onMemberAdded?.();
      } catch {
        console.error("Failed to add existing members");
      }
    };

    const handleSelectAll = () => {
      const filteredMembers = getFilteredExistingMembers();
      if (selectedMembers.size === filteredMembers.length) {
        setSelectedMembers(new Set());
      } else {
        setSelectedMembers(new Set(filteredMembers.map((m) => m.metadata!.id!)));
      }
    };

    const getFilteredExistingMembers = () => {
      if (!memberSearchTerm) return existingMembers;

      return existingMembers.filter(
        (member) =>
          member.spec?.displayName?.toLowerCase().includes(memberSearchTerm.toLowerCase()) ||
          member.metadata?.email?.toLowerCase().includes(memberSearchTerm.toLowerCase()),
      );
    };

    return (
      <div
        className={`bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 ${className}`}
      >
        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            <p className="text-sm">{error instanceof Error ? error.message : "Failed to fetch data"}</p>
          </div>
        )}

        <div className="flex items-center justify-between mb-4">
          <div>
            <Text className="font-semibold text-zinc-900 dark:text-white mb-1">Add members</Text>
          </div>
        </div>

        {/* Organization Members Content */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <InputGroup>
              <Input
                name="member-search"
                placeholder="Search members..."
                aria-label="Search members"
                className="w-xs"
                value={memberSearchTerm}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMemberSearchTerm(e.target.value)}
              />
            </InputGroup>
            <div className="flex items-center gap-2">
              <Button
                outline
                className="flex items-center gap-2 text-sm"
                onClick={handleSelectAll}
                disabled={loadingMembers || getFilteredExistingMembers().length === 0}
              >
                <Icon name="select_all" size="sm" />
                {selectedMembers.size === getFilteredExistingMembers().length ? "Deselect All" : "Select All"}
              </Button>
              <Button
                color="blue"
                className="flex items-center gap-2 text-sm"
                onClick={handleExistingMembersSubmit}
                disabled={selectedMembers.size === 0 || isInviting}
              >
                <Icon name="add" size="sm" />
                {isInviting
                  ? "Adding..."
                  : `Add ${selectedMembers.size} member${selectedMembers.size === 1 ? "" : "s"}`}
              </Button>
            </div>
          </div>

          {loadingMembers ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-zinc-500 dark:text-zinc-400">Loading members...</p>
            </div>
          ) : (
            <div className="max-h-96 overflow-y-auto border border-zinc-200 dark:border-zinc-700 rounded-lg">
              {getFilteredExistingMembers().length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-zinc-500 dark:text-zinc-400">
                    {memberSearchTerm
                      ? "No members found matching your search"
                      : "All organization members are already in this group"}
                  </p>
                </div>
              ) : (
                <div className="divide-y divide-zinc-200 dark:divide-zinc-700">
                  {getFilteredExistingMembers().map((member) => (
                    <div
                      key={member.metadata!.id!}
                      className="p-3 flex items-center gap-3 hover:bg-zinc-50 dark:hover:bg-zinc-800"
                    >
                      <Checkbox
                        checked={selectedMembers.has(member.metadata!.id!)}
                        onChange={(checked) => {
                          setSelectedMembers((prev) => {
                            const newSet = new Set(prev);
                            if (checked) {
                              newSet.add(member.metadata!.id!);
                            } else {
                              newSet.delete(member.metadata!.id!);
                            }
                            return newSet;
                          });
                        }}
                      />
                      <Avatar
                        src={member.spec?.accountProviders?.[0]?.avatarUrl}
                        initials={member.spec?.displayName?.charAt(0) || "U"}
                        className="size-8"
                      />
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-zinc-900 dark:text-white truncate">
                          {member.spec?.displayName || member.metadata!.id!}
                        </div>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
                          {member.metadata?.email || `${member.metadata!.id!}@email.placeholder`}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    );
  },
);

AddMembersSectionComponent.displayName = "AddMembersSection";

export const AddMembersSection = AddMembersSectionComponent;
