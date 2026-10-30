import { useQuery } from "@tanstack/react-query";

import { factoriesListFactoryCodexUsage, type FactoriesCodexMemberUsage } from "@/api-client";
import { Button } from "@/components/ui/button";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { CodexQuotaWindow } from "./CodexQuotaWindow";

export function WorkspaceCodexMembers({ organizationId, factoryId }: { organizationId: string; factoryId: string }) {
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
      <h2 className="text-[13px] font-medium">Connected members</h2>
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
          <ul className="divide-y divide-border">
            {query.data.map((member) => (
              <ConnectedMember key={member.userId} member={member} />
            ))}
          </ul>
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
    <li className="grid gap-3 py-3 first:pt-0 last:pb-0 sm:grid-cols-[minmax(0,1fr)_minmax(0,2fr)] sm:gap-6">
      <div className="min-w-0 text-[13px]">
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
      </div>
      <div className="space-y-3 text-[13px]">
        {member.observedAt && limits.length > 0 ? (
          limits.map((limit) => (
            <div key={limit.limitId} className="space-y-3">
              <p className="text-xs text-muted-foreground">
                {limit.limitName || limit.limitId || "Codex"}
                {limit.planType ? ` · ${limit.planType}` : ""}
              </p>
              {limit.primary ? <CodexQuotaWindow window={limit.primary} label="Primary window" /> : null}
              {limit.secondary ? <CodexQuotaWindow window={limit.secondary} label="Secondary window" /> : null}
            </div>
          ))
        ) : (
          <p className="text-xs text-muted-foreground">Usage unavailable.</p>
        )}
      </div>
    </li>
  );
}
