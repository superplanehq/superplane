import { forwardRef, useImperativeHandle, useMemo, useState } from "react";
import { Avatar } from "../../../components/Avatar/avatar";
import { Icon } from "../../../components/Icon";
import { Input } from "../../../components/Input/input";
import { useAddUserToGroup, useOrganizationGroupUsers, useOrganizationUsers } from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/ui/checkbox";
import { showErrorToast } from "@/utils/toast";

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
    } = useOrganizationUsers(organizationId, true);
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
        showErrorToast("Failed to add existing members");
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
        className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 ${className}`}
      >
        {error && (
          <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded mb-4">
            <p className="text-sm">{error instanceof Error ? error.message : "Failed to fetch data"}</p>
          </div>
        )}

        <div className="space-y-4">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="relative w-full md:max-w-sm">
              <Icon
                name="search"
                size="sm"
                className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none"
              />
              <Input
                name="member-search"
                placeholder="Search members..."
                aria-label="Search members"
                className="w-full pl-9"
                value={memberSearchTerm}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMemberSearchTerm(e.target.value)}
              />
            </div>
            <div className="flex items-center justify-end">
              <Button
                className="flex items-center gap-2 text-sm"
                onClick={handleExistingMembersSubmit}
                disabled={selectedMembers.size === 0 || isInviting}
              >
                <Icon name="plus" size="sm" />
                {isInviting
                  ? "Adding..."
                  : `Add ${selectedMembers.size} member${selectedMembers.size === 1 ? "" : "s"}`}
              </Button>
            </div>
          </div>

          {loadingMembers ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-gray-500 dark:text-gray-400">Loading members...</p>
            </div>
          ) : (
            <div className="max-h-96 overflow-y-auto">
              {getFilteredExistingMembers().length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    {memberSearchTerm
                      ? "No members found matching your search"
                      : "All organization members are already in this group"}
                  </p>
                </div>
              ) : (
                <div className="divide-y divide-gray-200 dark:divide-gray-700">
                  {getFilteredExistingMembers().map((member) => (
                    <div
                      key={member.metadata!.id!}
                      className="p-3 flex items-center gap-3 hover:bg-gray-50 dark:hover:bg-gray-800"
                    >
                      <Checkbox
                        checked={selectedMembers.has(member.metadata!.id!)}
                        onCheckedChange={(checked) => {
                          const isChecked = checked === true;
                          setSelectedMembers((prev) => {
                            const newSet = new Set(prev);
                            if (isChecked) {
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
                        <div className="text-sm font-medium text-gray-800 dark:text-white truncate">
                          {member.spec?.displayName || member.metadata!.id!}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 truncate">
                          {member.metadata?.email || "Service Account"}
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
