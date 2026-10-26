import type { ReactNode } from "react";
import { Avatar } from "@/components/Avatar/avatar";
import { Badge } from "@/components/Badge/badge";
import { TableCell, TableRow } from "@/components/Table/table";

const cellClassName =
  "block !border-0 !p-0 @min-[44rem]/members:table-cell @min-[44rem]/members:!border-b @min-[44rem]/members:!border-border @min-[44rem]/members:!px-4 @min-[44rem]/members:!py-3";

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
    <TableRow className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-2 border-b border-border px-3 py-3 last:border-0 @min-[44rem]/members:table-row @min-[44rem]/members:border-0 @min-[44rem]/members:p-0 @min-[44rem]/members:last:[&>td]:!border-b-0">
      <TableCell className={`${cellClassName} min-w-0`}>{identity}</TableCell>
      <TableCell className={`${cellClassName} col-start-1 row-start-2 !pl-11 @min-[44rem]/members:!pl-4`}>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          {role}
          <span className="@min-[44rem]/members:hidden">{status}</span>
        </div>
      </TableCell>
      <TableCell className={`${cellClassName} !hidden @min-[44rem]/members:!table-cell`}>{status}</TableCell>
      <TableCell className={`${cellClassName} col-start-2 row-span-2 row-start-1 self-center align-middle`}>
        <div className="flex items-center justify-end gap-1">{actions}</div>
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
  details,
}: {
  name?: string;
  email: string;
  initials: string;
  avatar?: string;
  isOwner?: boolean;
  details?: ReactNode;
}) {
  return (
    <div className="flex min-w-0 items-start gap-3">
      <span className="shrink-0">
        <Avatar src={avatar} initials={initials} className="size-8 shrink-0" />
      </span>
      <div className="min-w-0 space-y-0.5 @min-[44rem]/members:space-y-1">
        {name && (
          <div className="flex flex-wrap items-center gap-2">
            <span className="min-w-0 break-words font-medium leading-5 [overflow-wrap:anywhere] @min-[44rem]/members:leading-6">
              {name}
            </span>
            {isOwner && <Badge color="yellow">Owner</Badge>}
          </div>
        )}
        <p
          className={`whitespace-normal break-all ${name ? "text-xs leading-4 text-muted-foreground @min-[44rem]/members:leading-6" : "text-sm leading-5 @min-[44rem]/members:leading-6"}`}
        >
          {email}
        </p>
        {details}
      </div>
    </div>
  );
}

const memberStatusStyles = {
  active: { label: "Active", color: "green", dot: "bg-green-600 dark:bg-green-400" },
  pending: { label: "Pending", color: "yellow", dot: "bg-amber-600 dark:bg-amber-400" },
  expired: { label: "Expired", color: "zinc", dot: "bg-muted-foreground" },
} as const;

export function MemberStatus({ status }: { status: keyof typeof memberStatusStyles }) {
  const { label, color, dot } = memberStatusStyles[status];
  return (
    <>
      <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground @min-[44rem]/members:hidden">
        <span className={`size-1.5 shrink-0 rounded-full ${dot}`} aria-hidden="true" />
        {label}
      </span>
      <span className="hidden @min-[44rem]/members:inline-flex">
        <Badge color={color}>{label}</Badge>
      </span>
    </>
  );
}
