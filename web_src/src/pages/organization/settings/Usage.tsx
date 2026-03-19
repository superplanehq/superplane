import { useMemo } from "react";
import { Navigate } from "react-router-dom";
import { Activity, Database, Gauge, Layers3, Users } from "lucide-react";
import type { OrganizationsDescribeUsageResponse, OrganizationsOrganizationLimits } from "@/api-client/types.gen";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useOrganizationUsage } from "@/hooks/useOrganizationData";
import { isUsagePageForced } from "@/lib/env";
import { EmptyState } from "@/ui/emptyState";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";

interface UsageProps {
  organizationId: string;
}

type LimitCard = {
  label: string;
  value: string;
  icon: typeof Layers3;
  description: string;
};

const UNLIMITED_VALUE = "-1";

export function Usage({ organizationId }: UsageProps) {
  usePageTitle(["Usage"]);

  const { data, isLoading, error } = useOrganizationUsage(organizationId);
  const forceUsagePage = isUsagePageForced();
  const isPreviewMode = forceUsagePage && data?.enabled !== true;

  const usageCards = useMemo(() => buildLimitCards(data?.limits), [data?.limits]);
  const eventUsage = useMemo(() => buildEventUsage(data), [data]);
  const canvasUsage = useMemo(() => buildCanvasUsage(data), [data]);

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
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
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
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

  return (
    <div className="pt-6 space-y-6">
      <Alert>
        <Gauge className="h-4 w-4" />
        <AlertTitle>{isPreviewMode ? "Usage preview mode" : "Usage service connected"}</AlertTitle>
        <AlertDescription>
          {isPreviewMode
            ? "Showing the organization usage page in local development without a configured usage service."
            : data.statusMessage || "Organization usage is being tracked by the configured service."}
        </AlertDescription>
      </Alert>

      <div className="grid gap-4 md:grid-cols-2">
        <UsageMetricCard
          title="Canvases"
          value={canvasUsage.value}
          subtitle={canvasUsage.subtitle}
          progress={canvasUsage.progress}
          icon={Layers3}
        />
        <UsageMetricCard
          title="Event Budget"
          value={eventUsage.value}
          subtitle={eventUsage.subtitle}
          progress={eventUsage.progress}
          icon={Activity}
        />
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
        <div className="mb-4">
          <h2 className="text-base font-medium text-gray-900 dark:text-white">Limits</h2>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {usageCards.map((card) => (
            <div
              key={card.label}
              className="rounded-lg border border-gray-200 bg-slate-50 px-4 py-3 dark:border-gray-700 dark:bg-slate-900"
            >
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
  icon: typeof Gauge;
}) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
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
          <div className="h-full rounded-full bg-sky-500 transition-[width]" style={{ width: `${progress}%` }} />
        </div>
      )}
    </div>
  );
}

function buildCanvasUsage(data: OrganizationsDescribeUsageResponse | null | undefined) {
  const used = data?.usage?.canvases ?? 0;
  const limit = data?.limits?.maxCanvases;
  const limitLabel = formatNumericLimit(limit);

  return {
    value: `${formatNumber(used)} / ${limitLabel}`,
    subtitle: isUnlimitedNumber(limit)
      ? "This organization can create unlimited canvases."
      : "Active canvases tracked against the configured plan limit.",
    progress: percentage(used, limit),
  };
}

function buildEventUsage(data: OrganizationsDescribeUsageResponse | null | undefined) {
  const level = data?.usage?.eventBucketLevel ?? 0;
  const capacity = data?.usage?.eventBucketCapacity;
  const lastUpdatedAt = data?.usage?.eventBucketLastUpdatedAt;
  const isUnlimited = typeof capacity === "number" && capacity === -1;
  const value = isUnlimited ? "Unlimited" : `${formatNumber(level)} / ${formatNumber(capacity ?? 0)}`;
  const subtitle = lastUpdatedAt
    ? `Leaky bucket level last updated ${new Date(lastUpdatedAt).toLocaleString()}.`
    : "Rolling event usage for the current 30-day window.";

  return {
    value,
    subtitle,
    progress: isUnlimited ? null : percentage(level, capacity),
  };
}

function buildLimitCards(limits: OrganizationsOrganizationLimits | undefined): LimitCard[] {
  return [
    {
      label: "Nodes per canvas",
      value: formatNumericLimit(limits?.maxNodesPerCanvas),
      icon: Layers3,
      description: "Maximum nodes allowed on a single canvas.",
    },
    {
      label: "Members",
      value: formatNumericLimit(limits?.maxUsers),
      icon: Users,
      description: "Maximum users allowed in the organization plan.",
    },
    {
      label: "Integrations",
      value: formatNumericLimit(limits?.maxIntegrations),
      icon: Database,
      description: "Maximum connected integrations for the organization.",
    },
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
      description: "Rolling 30-day event allowance used by the leaky bucket.",
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

function formatNumericLimit(value: number | undefined) {
  if (value === undefined) {
    return "-";
  }

  if (isUnlimitedNumber(value)) {
    return "Unlimited";
  }

  return formatNumber(value);
}

function formatStringLimit(value: string | undefined) {
  if (!value) {
    return "-";
  }

  if (value === UNLIMITED_VALUE) {
    return "Unlimited";
  }

  return new Intl.NumberFormat().format(Number(value));
}

function formatDaysLimit(value: number | undefined) {
  if (value === undefined) {
    return "-";
  }

  if (isUnlimitedNumber(value)) {
    return "Unlimited";
  }

  if (value === 1) {
    return "1 day";
  }

  return `${formatNumber(value)} days`;
}

function isUnlimitedNumber(value: number | undefined) {
  return value === -1;
}
