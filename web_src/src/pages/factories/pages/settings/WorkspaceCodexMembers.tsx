import { useQuery } from "@tanstack/react-query";
import { EllipsisVertical } from "lucide-react";

import { factoriesListFactoryCodexUsage, type FactoriesCodexMemberUsage } from "@/api-client";
import { Avatar } from "@/components/Avatar/avatar";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/Table/table";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { getNameInitials } from "@/lib/nameInitials";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { settingsTableCardClassName } from "@/pages/organization/settings/settingsPageStyles";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { CodexQuotaRemaining, CodexQuotaWindow } from "./CodexQuotaWindow";

export function WorkspaceCodexMembers({
  organizationId,
  factoryId,
  canUpdate,
  defaultOwnerId,
  onChangeDefaultOwner,
}: {
  organizationId: string;
  factoryId: string;
  canUpdate: boolean;
  defaultOwnerId?: string;
  onChangeDefaultOwner: (ownerId: string) => void;
}) {
  const query = useQuery({
    queryKey: ["factories", organizationId, factoryId, "codex-usage"],
    queryFn: async ({ signal }) => {
      const response = await factoriesListFactoryCodexUsage(
        withOrganizationHeader({ organizationId, path: { factoryId }, signal }),
      );
      if (!response.data) throw new Error("Could not load connected ChatGPT members.");
      return response.data.members ?? [];
    },
  });
  const { canAct } = usePermissions();
  const organizationUsers = useOrganizationUsers(organizationId, false, canAct("members", "read"));
  const avatarsByUserId = new Map(
    (organizationUsers.data ?? []).flatMap((user) => {
      const userId = user.metadata?.id;
      const avatarUrl = user.status?.accountProviders?.[0]?.avatarUrl;
      return userId && avatarUrl ? [[userId, avatarUrl] as const] : [];
    }),
  );

  return (
    <section aria-label="Connected ChatGPT members" className="space-y-3">
      {query.isPending ? (
        <p role="status" className="text-xs text-muted-foreground">
          Loading connected members...
        </p>
      ) : null}
      {query.isError ? (
        <div className="space-y-2">
          <p role="alert" className="text-xs text-muted-foreground">
            Could not load connected ChatGPT members.
          </p>
          <Button type="button" variant="outline" size="sm" onClick={() => void query.refetch()}>
            Try again
          </Button>
        </div>
      ) : null}
      {query.isSuccess && query.data.length === 0 ? (
        <p role="status" className="text-xs text-muted-foreground">
          No members have a connected ChatGPT account.
        </p>
      ) : null}
      {query.data && query.data.length > 0 ? (
        <>
          <div className={`${settingsTableCardClassName} @container/members`}>
            <Table
              dense
              className="!mx-0 !w-full !whitespace-normal [&>div]:!w-full [&>div]:!px-0 [&_table]:!w-full [&_table]:min-w-[640px]"
            >
              <TableHead>
                <TableRow>
                  <TableHeader scope="col" className="!px-4 !py-3 !font-normal text-muted-foreground">
                    Member
                  </TableHeader>
                  <TableHeader scope="col" className="!px-4 !py-3 !font-normal text-muted-foreground">
                    Usage
                  </TableHeader>
                  <TableHeader scope="col" className="w-12 !px-2 !py-3 !font-normal text-muted-foreground">
                    <span className="sr-only">Actions</span>
                  </TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {query.data.map((member) => (
                  <ConnectedMember
                    key={member.userId}
                    member={member}
                    avatarUrl={member.userId ? avatarsByUserId.get(member.userId) : undefined}
                    canUpdate={canUpdate}
                    defaultOwnerId={defaultOwnerId}
                    onChangeDefaultOwner={onChangeDefaultOwner}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
          <p className="text-xs text-muted-foreground">
            Usage details show the last saved snapshot. SuperPlane checks usage during sign-in and Codex runs.
          </p>
        </>
      ) : null}
    </section>
  );
}

function ConnectedMember({
  member,
  avatarUrl,
  canUpdate,
  defaultOwnerId,
  onChangeDefaultOwner,
}: {
  member: FactoriesCodexMemberUsage;
  avatarUrl?: string;
  canUpdate: boolean;
  defaultOwnerId?: string;
  onChangeDefaultOwner: (ownerId: string) => void;
}) {
  const limits = member.limits ?? [];
  const name = member.displayName || member.email || "Workspace member";
  return (
    <TableRow className="last:[&>td]:!border-b-0">
      <TableCell className="min-w-[220px] !px-4 !py-3">
        <div className="flex min-w-0 items-center gap-3">
          <Avatar src={avatarUrl} initials={getNameInitials(name)} className="size-8 shrink-0" />
          <div className="min-w-0 space-y-0.5 @min-[44rem]/members:space-y-1">
            <p className="break-words font-medium leading-5 [overflow-wrap:anywhere] @min-[44rem]/members:leading-6">
              {name}
            </p>
            {member.displayName && member.email ? (
              <p className="whitespace-normal break-all text-xs leading-4 text-muted-foreground @min-[44rem]/members:leading-6">
                {member.email}
              </p>
            ) : null}
          </div>
        </div>
      </TableCell>
      <TableCell className="min-w-[220px] !px-4 !py-3 align-middle text-center">
        <UsageSummary limits={limits} />
      </TableCell>
      <TableCell className="w-12 !px-2 !py-2 text-right">
        {member.userId ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon-sm" aria-label={`Options for ${name}`}>
                <EllipsisVertical className="size-4" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-72">
              <DropdownMenuLabel>Subscription details</DropdownMenuLabel>
              {limits.length > 0 ? (
                limits.map((limit, index) => (
                  <div key={limit.limitId ?? index} className="px-2 py-1.5 text-sm">
                    <p className="font-medium">
                      {limit.limitName || limit.limitId || "Codex"}
                      {limit.planType ? ` · ${limit.planType}` : ""}
                    </p>
                    <div className="mt-2 space-y-3 text-muted-foreground">
                      {limit.primary ? <CodexQuotaWindow window={limit.primary} label="Primary window" /> : null}
                      {limit.secondary ? <CodexQuotaWindow window={limit.secondary} label="Secondary window" /> : null}
                      {!limit.primary && !limit.secondary ? <p>Usage unavailable</p> : null}
                    </div>
                  </div>
                ))
              ) : (
                <p className="px-2 py-1.5 text-sm text-muted-foreground">Usage unavailable</p>
              )}
              {member.observedAt ? (
                <p className="px-2 py-1.5 text-xs text-muted-foreground">
                  Last updated {new Date(member.observedAt).toLocaleString()}
                </p>
              ) : null}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                disabled={!canUpdate || member.userId === defaultOwnerId}
                onSelect={() => handleSetDefaultOwner(member, onChangeDefaultOwner)}
              >
                {member.userId === defaultOwnerId ? "Default task owner" : "Set as default task owner"}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </TableCell>
    </TableRow>
  );
}

function UsageSummary({ limits }: { limits: FactoriesCodexMemberUsage["limits"] }) {
  const windows = (limits ?? []).flatMap((limit, index) => [
    limit.primary ? { key: `${limit.limitId ?? index}-primary`, window: limit.primary, label: "Primary window" } : null,
    limit.secondary
      ? { key: `${limit.limitId ?? index}-secondary`, window: limit.secondary, label: "Secondary window" }
      : null,
  ]);
  if (windows.length === 0) return <p className="text-sm text-muted-foreground">Usage unavailable</p>;

  return (
    <div className={`flex flex-col justify-center ${windows.length > 1 ? "gap-2" : "min-h-10"}`}>
      {windows.map((item) =>
        item ? (
          <CodexQuotaRemaining
            key={item.key}
            window={item.window}
            label={item.label}
            showWindowLabel={windows.length > 1}
          />
        ) : null,
      )}
    </div>
  );
}

function handleSetDefaultOwner(member: FactoriesCodexMemberUsage, onChangeDefaultOwner: (ownerId: string) => void) {
  if (!member.userId) return;
  onChangeDefaultOwner(member.userId);
}
