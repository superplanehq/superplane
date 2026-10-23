import type { ReactNode } from "react";
import { Avatar } from "@/components/Avatar/avatar";
import { Badge } from "@/components/Badge/badge";
import { TableCell, TableRow } from "@/components/Table/table";

const cellClassName =
  "block !border-0 !p-0 @min-[44rem]/members:table-cell @min-[44rem]/members:!border-b @min-[44rem]/members:!border-border @min-[44rem]/members:!px-4 @min-[44rem]/members:!py-4";

export function MemberTableRow({
  identity,
  role,
  status,
  actions,
}: {
  identity: ReactNode;
  role: ReactNode;
  status: ReactNode;
  actions: ReactNode;
}) {
  return (
    <TableRow className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-2 border-b border-border px-4 py-4 last:border-0 @min-[44rem]/members:table-row @min-[44rem]/members:border-0 @min-[44rem]/members:p-0 @min-[44rem]/members:last:[&>td]:!border-b-0">
      <TableCell className={`${cellClassName} col-span-2 min-w-0`}>{identity}</TableCell>
      <TableCell className={`${cellClassName} col-start-1 row-start-2`}>
        <div className="flex flex-wrap items-center gap-2">
          {role}
          <span className="@min-[44rem]/members:hidden">{status}</span>
        </div>
      </TableCell>
      <TableCell className={`${cellClassName} !hidden @min-[44rem]/members:!table-cell`}>{status}</TableCell>
      <TableCell className={`${cellClassName} col-start-2 row-start-2 align-middle`}>
        <div className="flex h-full items-center justify-end gap-1">{actions}</div>
      </TableCell>
    </TableRow>
  );
}

export function MemberIdentity({
  name,
  email,
  initials,
  avatar,
  isOwner,
}: {
  name?: string;
  email: string;
  initials: string;
  avatar?: string;
  isOwner?: boolean;
}) {
  return (
    <div className="flex min-w-0 items-start gap-3">
      <span className="hidden @min-[44rem]/members:block">
        <Avatar src={avatar} initials={initials} className="size-8 shrink-0" />
      </span>
      <div className="min-w-0 space-y-1">
        {name && (
          <div className="flex flex-wrap items-center gap-2">
            <span className="min-w-0 break-words font-medium">{name}</span>
            {isOwner && <Badge color="yellow">Owner</Badge>}
          </div>
        )}
        <p className={`whitespace-normal break-all ${name ? "text-xs text-muted-foreground" : "text-sm"}`}>{email}</p>
      </div>
    </div>
  );
}
