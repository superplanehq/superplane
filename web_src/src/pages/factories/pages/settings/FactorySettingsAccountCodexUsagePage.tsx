import { CodexQuotaWindow } from "./CodexQuotaWindow";
import { type MeCodexUsageConnection } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useCodexUsage } from "@/hooks/useCodexUsage";
import { useFactoryCodexConnection } from "../onboarding/useFactoryCodexConnection";
import { FactoryCodexSubscriptionConnection } from "./FactoryCodexSubscriptionConnection";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function FactorySettingsAccountCodexUsagePage() {
  const { organizationId, factoryId } = useFactorySettingsLayout();
  const codex = useFactoryCodexConnection(organizationId, factoryId, true);
  const query = useCodexUsage(organizationId);

  return (
    <FactorySettingsPageFrame title="Codex">
      <FactorySettingsCard
        title="Your Codex connection"
        description="Connect once to use Codex across this organization’s workspaces."
      >
        <FactoryCodexSubscriptionConnection codex={codex} />
      </FactorySettingsCard>
      {query.isPending ? <p role="status">Loading Codex usage…</p> : null}
      {query.isError ? (
        <FactorySettingsCard title="Could not load Codex usage.">
          <Button variant="outline" onClick={() => void query.refetch()}>
            Try again
          </Button>
        </FactorySettingsCard>
      ) : null}
      {query.data ? <CodexUsageConnections connections={query.data} /> : null}
    </FactorySettingsPageFrame>
  );
}

export function CodexUsageConnections({ connections }: { connections: MeCodexUsageConnection[] }) {
  return connections.map((connection) => (
    <FactorySettingsCard
      key={connection.id}
      title={connection.accountLabel || "ChatGPT subscription"}
      description={
        connection.observedAt ? `Last updated ${new Date(connection.observedAt).toLocaleString()}` : undefined
      }
    >
      <div className="space-y-4 text-[13px]">
        {connection.status === "reconnect_required" ? (
          <p>Connect your ChatGPT subscription again to update usage.</p>
        ) : null}
        {connection.observedAt && connection.limits?.some((limit) => limit.primary || limit.secondary) ? (
          connection.limits.map((limit) => (
            <div key={limit.limitId} className="space-y-3">
              <p className="font-medium">
                {limit.limitName || limit.limitId || "Codex"}
                {limit.planType ? ` · ${limit.planType}` : ""}
              </p>
              {limit.primary ? <CodexQuotaWindow window={limit.primary} label="Primary window" /> : null}
              {limit.secondary ? <CodexQuotaWindow window={limit.secondary} label="Secondary window" /> : null}
            </div>
          ))
        ) : (
          <p>Usage unavailable. Sign in to ChatGPT or run Codex in SuperPlane to collect usage.</p>
        )}
        <p className="text-[12px] text-muted-foreground">
          SuperPlane checks usage when you sign in and during Codex runs. This page shows the last saved snapshot.
        </p>
      </div>
    </FactorySettingsCard>
  ));
}
