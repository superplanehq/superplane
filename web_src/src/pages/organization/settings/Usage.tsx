import { useMemo } from "react";
import { Navigate } from "react-router-dom";
import { Activity, Bot, Gauge, type LucideIcon } from "lucide-react";
import type { OrganizationsDescribeUsageResponse, OrganizationsOrganizationLimits } from "@/api-client/types.gen";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useOrganizationUsage } from "@/hooks/useOrganizationData";
import { isUsagePageForced } from "@/lib/env";
import { EmptyState } from "@/ui/emptyState";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { settingsCardClassName, settingsInnerMetricCardClassName } from "./settingsPageStyles";

interface UsageProps {
  organizationId: string;
}

type LimitCard = {
  label: string;
  value: string;
  icon: LucideIcon;
  description: string;
};

const UNLIMITED_VALUE = "-1";

export function Usage({ organizationId }: UsageProps) {
  usePageTitle(["Usage"]);

  // Usage can change right before the user opens this page, so force a fetch and hide stale cached data while it runs.
  const { data, isLoading, isFetching, error } = useOrganizationUsage(organizationId, true, {
    staleTime: 0,
    gcTime: 0,
    refetchOnMount: "always",
  });
  const forceUsagePage = isUsagePageForced();
  const isUsageLoading = isLoading || isFetching;

  useReportPageReady(!isUsageLoading, {
    failed: !!error,
  });

  if (isUsageLoading) {
    return (
      <div className="pt-6">
        <div className={settingsCardClassName}>
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading usage...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="pt-6">
        <Alert variant="destructive">
          <Gauge className="h-4 w-4" />
          <AlertTitle>Unable to load usage</AlertTitle>
          <AlertDescription>{error instanceof Error ? error.message : "Unknown error"}</AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="pt-6">
        <div className={settingsCardClassName}>
          <EmptyState
            icon={Gauge}
            title="Usage data unavailable"
            description="SuperPlane could not load usage information for this organization."
          />
        </div>
      </div>
    );
  }

  if (!data.enabled && !forceUsagePage) {
    return <Navigate to={`/${organizationId}/settings/general`} replace />;
  }

  return <UsageContent data={data} isPreviewMode={forceUsagePage && data.enabled !== true} />;
}

function UsageContent({ data, isPreviewMode }: { data: OrganizationsDescribeUsageResponse; isPreviewMode: boolean }) {
  const usageCards = useMemo(() => buildLimitCards(data.limits), [data.limits]);
  const eventUsage = useMemo(() => buildEventUsage(data), [data]);
  const agentTokenUsage = useMemo(() => buildAgentTokenUsage(data), [data]);

  return (
    <div className="pt-6 space-y-6">
      <div className={settingsCardClassName}>
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-white">
              {isPreviewMode ? "Usage preview mode" : "Usage tracking active"}
            </p>
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              {isPreviewMode
                ? "Showing the organization usage page in local development without configured usage tracking."
                : data.statusMessage || "Organization usage is being tracked for this organization."}
            </p>
          </div>
          <Gauge className="h-5 w-5 shrink-0 text-gray-500 dark:text-gray-400" />
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <UsageMetricCard
          title="Event Budget"
          value={eventUsage.value}
          subtitle={eventUsage.subtitle}
          progress={eventUsage.progress}
          icon={Activity}
        />
        <UsageMetricCard
          title="Agent Tokens"
          value={agentTokenUsage.value}
          subtitle={agentTokenUsage.subtitle}
          progress={agentTokenUsage.progress}
          icon={Bot}
        />
      </div>

      <div className={settingsCardClassName}>
        <div className="mb-4">
          <h2 className="text-base font-medium text-gray-900 dark:text-white">Limits</h2>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          {usageCards.map((card) => (
            <div key={card.label} className={settingsInnerMetricCardClassName}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-white">{card.label}</p>
                  <p className="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{card.value}</p>
                </div>
                <card.icon className="h-4 w-4 text-gray-500 dark:text-gray-400" />
              </div>
              <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">{card.description}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function UsageMetricCard({
  title,
  value,
  subtitle,
  progress,
  icon: Icon,
}: {
  title: string;
  value: string;
  subtitle: string;
  progress: number | null;
  icon: LucideIcon;
}) {
  return (
    <div className={settingsCardClassName}>
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-gray-900 dark:text-white">{title}</p>
          <p className="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{value}</p>
        </div>
        <Icon className="h-5 w-5 text-gray-500 dark:text-gray-400" />
      </div>
      <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">{subtitle}</p>
      {progress !== null && (
        <div className="mt-4 h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700">
          <div
            className="h-full rounded-full bg-sky-500 transition-[width] dark:bg-indigo-300"
            style={{ width: `${progress}%` }}
          />
        </div>
      )}
    </div>
  );
}

function buildBucketUsage(
  level: number,
  capacity: number | undefined,
  lastUpdatedAt: string | undefined,
  nextDecreaseAt: string | undefined,
  defaultSubtitle: string,
) {
  const displayedLevel = Math.max(0, Math.ceil(level));
  const isUnlimited = typeof capacity === "number" && capacity === -1;
  const value = `${formatNumber(displayedLevel)} / ${isUnlimited ? "∞" : formatNumber(capacity ?? 0)}`;

  let subtitle = defaultSubtitle;
  if (nextDecreaseAt) {
    subtitle = `Next usage decrease ${new Date(nextDecreaseAt).toLocaleString()}.`;
  } else if (lastUpdatedAt) {
    subtitle = `Last updated ${new Date(lastUpdatedAt).toLocaleString()}.`;
  }

  return {
    value,
    subtitle,
    progress: isUnlimited ? null : percentage(displayedLevel, capacity),
  };
}

function buildEventUsage(data: OrganizationsDescribeUsageResponse | null | undefined) {
  return buildBucketUsage(
    data?.usage?.eventBucketLevel ?? 0,
    data?.usage?.eventBucketCapacity,
    data?.usage?.eventBucketLastUpdatedAt,
    data?.usage?.nextEventBucketDecreaseAt,
    "Rolling event usage for the current 30-day window.",
  );
}

function buildAgentTokenUsage(data: OrganizationsDescribeUsageResponse | null | undefined) {
  return buildBucketUsage(
    data?.usage?.agentTokenBucketLevel ?? 0,
    data?.usage?.agentTokenBucketCapacity,
    data?.usage?.agentTokenBucketLastUpdatedAt,
    data?.usage?.nextAgentTokenBucketDecreaseAt,
    "Rolling agent token usage for the current 30-day window.",
  );
}

function buildLimitCards(limits: OrganizationsOrganizationLimits | undefined): LimitCard[] {
  return [
    {
      label: "Retention window",
      value: formatDaysLimit(limits?.retentionWindowDays),
      icon: Activity,
      description: "How long usage-related data remains available.",
    },
    {
      label: "Events per month",
      value: formatStringLimit(limits?.maxEventsPerMonth),
      icon: Gauge,
      description: "Rolling 30-day event allowance.",
    },
    {
      label: "Agent tokens per month",
      value: formatStringLimit(limits?.maxAgentTokensPerMonth),
      icon: Bot,
      description: "Rolling 30-day agent token allowance.",
    },
  ];
}

function percentage(value: number, max: number | undefined | null) {
  if (max === undefined || max === null || max <= 0 || max === -1) {
    return null;
  }

  return Math.max(0, Math.min(100, (value / max) * 100));
}

function formatNumber(value: number) {
  return new Intl.NumberFormat().format(Math.round(value * 100) / 100);
}

function formatStringLimit(value: string | undefined) {
  if (!value) {
    return "-";
  }

  if (value === UNLIMITED_VALUE) {
    return "∞";
  }

  return new Intl.NumberFormat().format(Number(value));
}

function formatDaysLimit(value: number | undefined) {
  if (value === undefined) {
    return "-";
  }

  if (isUnlimitedNumber(value)) {
    return "∞";
  }

  if (value === 1) {
    return "1 day";
  }

  return `${formatNumber(value)} days`;
}

function isUnlimitedNumber(value: number | undefined) {
  return value === -1;
}
