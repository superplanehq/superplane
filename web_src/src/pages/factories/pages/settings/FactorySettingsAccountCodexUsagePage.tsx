import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { meListCodexUsage, type MeCodexUsageConnection, type MeCodexUsageWindow } from "@/api-client";
import { Button } from "@/components/ui/button";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryCodexConnection } from "../onboarding/useFactoryCodexConnection";
import { FactoryCodexSubscriptionConnection } from "./FactoryCodexSubscriptionConnection";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_CODEX_SUBSCRIPTIONS } from "@/lib/experimentalFeatures";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function FactorySettingsAccountCodexUsagePage() {
  const { organizationId, factoryId } = useFactorySettingsLayout();
  const { currentUserId } = usePermissions();
  const features = useExperimentalFeature(organizationId);
  const connectionEnabled = !features.isLoading && features.has(FEATURE_CODEX_SUBSCRIPTIONS);
  const codex = useFactoryCodexConnection(organizationId, factoryId, connectionEnabled);
  const query = useQuery({
    queryKey: ["me", organizationId, "codex-usage", currentUserId],
    queryFn: async () => {
      const response = await meListCodexUsage(withOrganizationHeader({ organizationId }));
      if (!response.data) throw new Error("Could not load Codex usage.");
      return response.data.connections ?? [];
    },
    enabled: Boolean(organizationId),
    refetchInterval: false,
    refetchOnWindowFocus: false,
  });

  return (
    <FactorySettingsPageFrame
      title="Codex usage"
      subtitle="Saved usage for your ChatGPT subscriptions. Usage outside SuperPlane can change the remaining quota."
    >
      {connectionEnabled ? <FactoryCodexSubscriptionConnection codex={codex} /> : null}
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
  if (!connections.length) {
    return (
      <FactorySettingsCard
        title="No ChatGPT subscription connected"
        description="Connect your ChatGPT subscription to see Codex usage here."
      />
    );
  }

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
              {limit.primary ? <QuotaWindow window={limit.primary} label="Primary window" /> : null}
              {limit.secondary ? <QuotaWindow window={limit.secondary} label="Secondary window" /> : null}
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

function QuotaWindow({ window, label }: { window: MeCodexUsageWindow; label: string }) {
  const [now, setNow] = useState(Date.now);
  const resetsAt = window.resetsAt ? new Date(window.resetsAt).getTime() : NaN;
  useEffect(() => {
    if (!Number.isFinite(resetsAt)) return;
    let timer: ReturnType<typeof setTimeout>;
    const scheduleReset = () => {
      const current = Date.now();
      setNow(current);
      if (resetsAt > current) {
        timer = setTimeout(scheduleReset, Math.min(resetsAt - current, 2147483647));
      }
    };
    scheduleReset();
    return () => clearTimeout(timer);
  }, [resetsAt]);
  const minutes = Number(window.windowDurationMins);
  const windowLabel =
    minutes > 0
      ? minutes % 1440 === 0
        ? `${minutes / 1440}-day window`
        : minutes % 60 === 0
          ? `${minutes / 60}-hour window`
          : `${minutes}-minute window`
      : label;
  const expired = resetsAt <= Math.max(now, Date.now());
  const remaining = window.usedPercent === undefined ? undefined : Math.max(0, Math.min(100, 100 - window.usedPercent));

  return (
    <div className="space-y-1.5">
      <div className="flex justify-between gap-3">
        <span>{windowLabel}</span>
        <span className="font-medium">
          {expired ? "Update needed" : remaining === undefined ? "Usage unavailable" : `${remaining}% remaining`}
        </span>
      </div>
      {!expired && remaining !== undefined ? (
        <div
          role="progressbar"
          aria-label={`${windowLabel} remaining`}
          aria-valuenow={remaining}
          aria-valuemin={0}
          aria-valuemax={100}
          className="h-2 overflow-hidden rounded-full bg-muted"
        >
          <div className="h-full bg-primary" style={{ width: `${remaining}%` }} />
        </div>
      ) : null}
      {window.resetsAt ? (
        <p className="text-[12px] text-muted-foreground">
          {expired ? "Reset time passed" : "Resets"}: {new Date(window.resetsAt).toLocaleString()}
        </p>
      ) : null}
    </div>
  );
}
