import { Avatar } from "@/components/Avatar/avatar";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getNameInitials } from "@/lib/nameInitials";
import { Switch } from "@/ui/switch";

import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import {
  type SettingsRedesignMember,
  type SettingsRedesignRoleOption,
  useSettingsRedesignOrganizationMembers,
} from "./useSettingsRedesignOrganizationMembers";

export function SettingsRedesignOrganizationMembersPage() {
  const members = useSettingsRedesignOrganizationMembers();
  usePageTitle(["Members"]);

  return (
    <FactorySettingsPageFrame title="Members" subtitle="Invite people and manage organization access.">
      <div className="space-y-6">
        <InviteLinkSection members={members} />
        <div className="space-y-3 border-t border-border pt-6">
          <h2 className="text-[13px] font-medium text-foreground">Members ({members.members.length})</h2>
          {members.usersError ? <p className="text-[13px] text-destructive">{members.usersError}</p> : null}
          {members.usersLoading ? <p className="text-[13px] text-muted-foreground">Loading members...</p> : null}
          {!members.usersLoading ? <MemberList members={members} /> : null}
        </div>
      </div>
    </FactorySettingsPageFrame>
  );
}

function InviteLinkSection({ members }: { members: ReturnType<typeof useSettingsRedesignOrganizationMembers> }) {
  return (
    <div className="space-y-3" data-testid="settings-redesign-invite-link">
      <div className="flex items-start justify-between gap-6">
        <div>
          <h2 className="text-[13px] font-medium text-foreground">Invite link</h2>
          <p className="text-[12px] text-muted-foreground">People with this link can join the organization.</p>
        </div>
        <PermissionTooltip
          allowed={members.canManageInvite || members.permissionsLoading}
          message="You do not have permission to manage invite links."
        >
          <Switch
            checked={members.inviteEnabled}
            onCheckedChange={(enabled) => void members.toggleInvite(enabled)}
            disabled={members.inviteLoading || members.inviteBusy || !members.canManageInvite}
            aria-label="Toggle invite link"
          />
        </PermissionTooltip>
      </div>
      {members.inviteEnabled && members.inviteUrl ? (
        <div className="flex flex-wrap items-center gap-3">
          <Input readOnly value={members.inviteUrl} className="min-w-0 flex-1" />
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!members.canManageInvite || members.inviteBusy}
            onClick={() => void members.copyInvite()}
          >
            Copy link
          </Button>
        </div>
      ) : (
        <p className="text-[13px] text-muted-foreground">Invite link is off.</p>
      )}
    </div>
  );
}

function MemberList({ members }: { members: ReturnType<typeof useSettingsRedesignOrganizationMembers> }) {
  if (members.members.length === 0) {
    return <p className="text-[13px] text-muted-foreground">No members yet.</p>;
  }

  return (
    <ul className="divide-y divide-border" data-testid="settings-redesign-members">
      {members.members.map((member) => {
        const isSelf = member.id === members.meId;
        const lastOwner = member.roleName === "org_owner" && members.ownerIds.size <= 1;
        return (
          <li key={member.id} className="flex items-center justify-between gap-3 py-3 first:pt-0">
            <div className="flex min-w-0 items-center gap-3">
              <Avatar initials={getNameInitials(member.name) || "?"} alt={member.name} className="size-8" />
              <div className="min-w-0">
                <p className="truncate text-[13px] font-medium text-foreground">{member.name}</p>
                <p className="truncate text-[12px] text-muted-foreground">{member.email}</p>
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <MemberRoleSelect
                member={member}
                roles={members.roles}
                disabled={isSelf || !members.canUpdateMembers}
                loading={members.rolesLoading}
                onRoleChange={(roleName) => void members.changeRole(member.id, roleName)}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={isSelf || lastOwner || !members.canDeleteMembers}
                onClick={() => void members.removeMember(member)}
              >
                Remove
              </Button>
            </div>
          </li>
        );
      })}
    </ul>
  );
}

function MemberRoleSelect({
  member,
  roles,
  disabled,
  loading,
  onRoleChange,
}: {
  member: SettingsRedesignMember;
  roles: SettingsRedesignRoleOption[];
  disabled: boolean;
  loading: boolean;
  onRoleChange: (roleName: string) => void;
}) {
  return (
    <Select value={member.roleName} disabled={disabled || loading} onValueChange={onRoleChange}>
      <SelectTrigger
        size="sm"
        aria-label={`Role for ${member.name}`}
        data-testid={`settings-redesign-member-role-${member.id}`}
        className="h-7 w-fit gap-1 px-2 py-0 pr-1.5 text-[13px] data-[size=sm]:h-7"
      >
        <SelectValue>{member.roleLabel}</SelectValue>
      </SelectTrigger>
      <SelectContent position="popper" align="end">
        {roles.map((role) => (
          <SelectItem key={role.name} value={role.name}>
            {role.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
