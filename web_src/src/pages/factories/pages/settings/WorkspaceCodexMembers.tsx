import { useQuery } from "@tanstack/react-query";

import { factoriesListFactoryCodexUsage, type FactoriesCodexMemberUsage } from "@/api-client";
import { Button } from "@/components/ui/button";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { CodexQuotaWindow } from "./CodexQuotaWindow";

export function WorkspaceCodexMembers({
  organizationId,
  factoryId,
  canUpdate,
  onChangeDefaultOwner,
}: {
  organizationId: string;
  factoryId: string;
  canUpdate: boolean;
  onChangeDefaultOwner?: () => void;
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

  return (
    <section aria-label="Connected ChatGPT members" className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-[13px] font-medium">Connected members</h2>
        {onChangeDefaultOwner ? (
          <Button type="button" variant="outline" size="sm" disabled={!canUpdate} onClick={onChangeDefaultOwner}>
            Change default task owner
          </Button>
        ) : null}
      </div>
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
          <div className="overflow-x-auto rounded-md border border-border">
            <table className="w-full min-w-[640px] text-left text-[13px]">
              <thead className="border-b border-border bg-muted/30 text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className="px-3 py-2.5 font-medium">
                    Member
                  </th>
                  <th scope="col" className="px-3 py-2.5 font-medium">
                    Subscription
                  </th>
                  <th scope="col" className="px-3 py-2.5 font-medium">
                    Usage
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {query.data.map((member) => (
                  <ConnectedMember key={member.userId} member={member} />
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-xs text-muted-foreground">
            Usage shows the last saved snapshot. SuperPlane checks usage during sign-in and Codex runs.
          </p>
        </>
      ) : null}
    </section>
  );
}

function ConnectedMember({ member }: { member: FactoriesCodexMemberUsage }) {
  const limits = member.limits?.filter((limit) => limit.primary || limit.secondary) ?? [];
  return (
    <tr>
      <td className="min-w-[220px] px-3 py-3 align-top">
        <p className="break-words font-medium [overflow-wrap:anywhere]">
          {member.displayName || member.email || "Workspace member"}
        </p>
        {member.displayName && member.email ? (
          <p className="mt-0.5 break-words text-xs text-muted-foreground [overflow-wrap:anywhere]">{member.email}</p>
        ) : null}
        {member.observedAt ? (
          <p className="mt-1 text-xs text-muted-foreground">
            Last updated {new Date(member.observedAt).toLocaleString()}
          </p>
        ) : null}
      </td>
      <td className="px-3 py-3 align-top text-muted-foreground">
        {limits.map((limit) => (
          <p key={limit.limitId}>
            {limit.limitName || limit.limitId || "Codex"}
            {limit.planType ? ` · ${limit.planType}` : ""}
          </p>
        ))}
        {limits.length === 0 ? "Usage unavailable" : null}
      </td>
      <td className="min-w-[260px] px-3 py-3 align-top">
        {member.observedAt && limits.length > 0 ? (
          <div className="space-y-2">
            {limits.flatMap((limit) => [
              limit.primary ? (
                <CodexQuotaWindow key={`${limit.limitId}-primary`} window={limit.primary} label="Primary window" />
              ) : null,
              limit.secondary ? (
                <CodexQuotaWindow
                  key={`${limit.limitId}-secondary`}
                  window={limit.secondary}
                  label="Secondary window"
                />
              ) : null,
            ])}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">Usage unavailable</p>
        )}
      </td>
    </tr>
  );
}
