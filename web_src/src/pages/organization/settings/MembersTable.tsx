import { Icon } from "@/components/Icon";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/Table/table";
import { MemberIdentity, MemberStatus, MemberTableRow } from "./MemberTableRow";
import { MemberInvitationRow } from "./MemberInvitationRow";
import { MemberOverflowMenu, MemberRoleSelect, type OrganizationMember } from "./MemberRowControls";
import type { InvitationManager } from "./useOrganizationInvitationManager";
import type { ComponentProps } from "react";

type MemberControls = Omit<ComponentProps<typeof MemberRoleSelect>, "member"> &
  Omit<ComponentProps<typeof MemberOverflowMenu>, "member">;

interface MembersTableProps {
  members: OrganizationMember[];
  manager: InvitationManager;
  controls: MemberControls;
  onShowLink: () => void;
  sort: { onSort: (key: keyof OrganizationMember) => void; icon: (key: keyof OrganizationMember) => string };
}

export function MembersTable({ members, manager, controls, onShowLink, sort }: MembersTableProps) {
  const invitations =
    manager.invitations.data?.invitations?.filter(
      (invitation) => invitation.state === "pending" || invitation.state === "expired",
    ) ?? [];
  return (
    <Table
      dense
      className="!mx-0 !overflow-visible !whitespace-normal [&>div]:!block [&>div]:!px-0 [&_table]:block [&_table]:w-full [&_table]:table-fixed @min-[44rem]/members:[&_table]:table"
    >
      <TableHead className="hidden @min-[44rem]/members:table-header-group">
        <TableRow>
          <MemberSortHeader label="Member" field="name" sort={sort} />
          <MemberSortHeader label="Role" field="role" sort={sort} className="w-36" />
          <MemberSortHeader label="Status" field="status" sort={sort} className="w-28" />
          <TableHeader className="w-16 !pr-4 text-right">
            <span className="sr-only">Actions</span>
          </TableHeader>
        </TableRow>
      </TableHead>
      <TableBody className="block @min-[44rem]/members:table-row-group">
        {members.map((member) => (
          <MemberTableRow
            key={member.id}
            identity={
              <MemberIdentity
                name={member.name}
                email={member.email}
                initials={member.initials}
                avatar={member.avatar}
                isOwner={member.isOwner}
              />
            }
            role={<MemberRoleSelect member={member} {...controls} />}
            status={<MemberStatus status="active" />}
            actions={<MemberOverflowMenu member={member} {...controls} />}
          />
        ))}
        {invitations.map((invitation) => (
          <MemberInvitationRow
            key={invitation.id}
            invitation={invitation}
            manager={manager}
            roles={controls.assignableRoles}
            onShowLink={onShowLink}
          />
        ))}
      </TableBody>
    </Table>
  );
}

function MemberSortHeader({
  label,
  field,
  sort,
  className = "",
}: {
  label: string;
  field: keyof OrganizationMember;
  sort: MembersTableProps["sort"];
  className?: string;
}) {
  return (
    <TableHeader className={`${className} !px-4`}>
      <button type="button" className="flex items-center gap-2" onClick={() => sort.onSort(field)}>
        {label}
        <Icon name={sort.icon(field)} size="sm" />
      </button>
    </TableHeader>
  );
}
