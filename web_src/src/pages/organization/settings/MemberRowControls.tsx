import { Icon } from "../../../components/Icon";
import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { settingsRowMenuClassName } from "./settingsPageStyles";

export interface OrganizationMember {
  id: string;
  name: string;
  email: string;
  role: string;
  roleName: string;
  initials: string;
  avatar?: string;
  type: "member";
  status: "active";
  isOwner: boolean;
}

export interface AssignableRoleOption {
  metadata?: { name?: string };
  spec?: { displayName?: string };
}

interface MemberRoleSelectProps {
  member: OrganizationMember;
  currentUserId?: string;
  canUpdateMembers: boolean;
  permissionsLoading: boolean;
  loadingRoles: boolean;
  assignableRoles: AssignableRoleOption[];
  onRoleChange: (memberId: string, roleName: string) => void;
}

export function MemberRoleSelect({
  member,
  currentUserId,
  canUpdateMembers,
  permissionsLoading,
  loadingRoles,
  assignableRoles,
  onRoleChange,
}: MemberRoleSelectProps) {
  const isSelf = currentUserId === member.id;
  const roleChangeAllowed = canUpdateMembers && !isSelf;
  const tooltipAllowed = roleChangeAllowed || (permissionsLoading && !isSelf);
  const tooltipMessage = isSelf
    ? "You can't change your own role."
    : "You don't have permission to update member roles.";

  return (
    <PermissionTooltip allowed={tooltipAllowed} message={tooltipMessage}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            className={cn("flex items-center gap-2 text-sm", settingsRowMenuClassName)}
            disabled={!roleChangeAllowed}
          >
            {member.role}
            <Icon name="chevron-down" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          {assignableRoles.map((role) => (
            <DropdownMenuItem
              key={role.metadata?.name}
              onClick={() => onRoleChange(member.id, role.metadata?.name || "")}
              disabled={loadingRoles || !roleChangeAllowed}
              className="flex flex-col items-start gap-1"
            >
              <span className="text-sm font-medium text-gray-800 dark:text-gray-100">
                {role.spec?.displayName || role.metadata?.name}
              </span>
            </DropdownMenuItem>
          ))}
          {loadingRoles && (
            <DropdownMenuItem disabled>
              <span className="text-sm text-gray-500 dark:text-gray-400">Loading roles...</span>
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </PermissionTooltip>
  );
}

interface MemberOverflowMenuProps {
  member: OrganizationMember;
  ownerCount: number;
  canUpdateMembers: boolean;
  canDeleteMembers: boolean;
  permissionsLoading: boolean;
  ownerChangePending: boolean;
  onOwnerChange: (member: OrganizationMember, isOwner: boolean) => void;
  onRemove: (member: OrganizationMember) => void;
}

export function MemberOverflowMenu({
  member,
  ownerCount,
  canUpdateMembers,
  canDeleteMembers,
  permissionsLoading,
  ownerChangePending,
  onOwnerChange,
  onRemove,
}: MemberOverflowMenuProps) {
  const isLastOwner = member.isOwner && ownerCount <= 1;
  const ownerActionAllowed = canUpdateMembers && !isLastOwner;
  const canSetOwner = canUpdateMembers && !member.isOwner;
  const canClearOwner = canUpdateMembers && member.isOwner && !isLastOwner;

  return (
    <PermissionTooltip
      allowed={canDeleteMembers || canUpdateMembers || permissionsLoading}
      message="You don't have permission to manage members."
    >
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            className={cn("flex items-center gap-2 text-sm disabled:opacity-50", settingsRowMenuClassName)}
            disabled={!canDeleteMembers && !canUpdateMembers}
          >
            <Icon name="ellipsis-vertical" size="sm" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          {canSetOwner && (
            <DropdownMenuItem
              className="flex items-center gap-1"
              onClick={() => onOwnerChange(member, true)}
              disabled={!ownerActionAllowed || ownerChangePending}
            >
              <Icon name="star" size="sm" />
              Set as owner
            </DropdownMenuItem>
          )}
          {canClearOwner && (
            <DropdownMenuItem
              className="flex items-center gap-1"
              onClick={() => onOwnerChange(member, false)}
              disabled={ownerChangePending}
            >
              <Icon name="star" size="sm" />
              Remove owner
            </DropdownMenuItem>
          )}
          {isLastOwner ? (
            <DropdownMenuItem disabled>
              <Icon name="x" size="sm" />
              <span className="ml-1">Cannot remove last owner</span>
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem
              className="flex items-center gap-1"
              onClick={() => onRemove(member)}
              disabled={!canDeleteMembers}
            >
              <Icon name="x" size="sm" />
              Remove
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </PermissionTooltip>
  );
}
