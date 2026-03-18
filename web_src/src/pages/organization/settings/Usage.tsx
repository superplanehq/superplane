import { useParams } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useOrganizationUsage } from "@/hooks/useOrganizationData";
import { Fieldset } from "@/components/Fieldset/fieldset";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface UsageBarProps {
  label: string;
  current: number;
  limit: number;
  isUnlimited: boolean;
}

function UsageBar({ label, current, limit, isUnlimited }: UsageBarProps) {
  const percentage = isUnlimited || limit === 0 ? 0 : Math.min((current / limit) * 100, 100);
  const isNearLimit = !isUnlimited && limit > 0 && percentage >= 80;
  const isAtLimit = !isUnlimited && limit > 0 && percentage >= 100;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium text-gray-700 dark:text-gray-300">{label}</span>
        <span className="text-gray-500 dark:text-gray-400">
          {isUnlimited ? (
            <span>{current.toLocaleString()} (unlimited)</span>
          ) : (
            <span>
              {current.toLocaleString()} / {limit.toLocaleString()}
            </span>
          )}
        </span>
      </div>
      {!isUnlimited && limit > 0 && (
        <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
          <div
            className={cn(
              "h-full rounded-full transition-all duration-300",
              isAtLimit ? "bg-red-500" : isNearLimit ? "bg-amber-500" : "bg-sky-500",
            )}
            style={{ width: `${percentage}%` }}
          />
        </div>
      )}
    </div>
  );
}

export function Usage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  usePageTitle(["Usage"]);

  const { data: usage, isLoading, error } = useOrganizationUsage(organizationId || "");

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-32">
        <p className="text-gray-500 dark:text-gray-400">Loading usage data...</p>
      </div>
    );
  }

  if (error || !usage) {
    return (
      <div className="flex justify-center items-center h-32">
        <p className="text-gray-500 dark:text-gray-400">Failed to load usage data.</p>
      </div>
    );
  }

  const limits = usage.effectiveLimits;
  const counters = usage.currentUsage;
  const isUnlimited = usage.isUnlimited ?? false;

  return (
    <div className="space-y-6 pt-6 text-left">
      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Usage Profile</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              {isUnlimited
                ? "This organization has unlimited resources."
                : `Active profile: ${usage.profileName || "default"}`}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {isUnlimited && <Badge variant="secondary">Unlimited</Badge>}
            {usage.hasOverrides && !isUnlimited && <Badge variant="outline">Custom Overrides</Badge>}
          </div>
        </div>
      </Fieldset>

      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-5">
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Resource Usage</h3>
        <UsageBar
          label="Canvases"
          current={Number(counters?.canvases ?? 0)}
          limit={limits?.maxCanvasesPerOrg ?? 0}
          isUnlimited={isUnlimited}
        />
        <UsageBar
          label="Members"
          current={Number(counters?.users ?? 0)}
          limit={limits?.maxUsersPerOrg ?? 0}
          isUnlimited={isUnlimited}
        />
        <UsageBar
          label="Integrations"
          current={Number(counters?.integrations ?? 0)}
          limit={limits?.maxIntegrationsPerOrg ?? 0}
          isUnlimited={isUnlimited}
        />
        <UsageBar
          label="Events this month"
          current={Number(counters?.eventsThisMonth ?? 0)}
          limit={limits?.maxEventsPerMonth ?? 0}
          isUnlimited={isUnlimited}
        />
      </Fieldset>

      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-4">
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Limits</h3>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <LimitRow label="Organizations per account" value={limits?.maxOrgsPerAccount} unlimited={isUnlimited} />
          <LimitRow label="Canvases per organization" value={limits?.maxCanvasesPerOrg} unlimited={isUnlimited} />
          <LimitRow label="Nodes per canvas" value={limits?.maxNodesPerCanvas} unlimited={isUnlimited} />
          <LimitRow label="Members per organization" value={limits?.maxUsersPerOrg} unlimited={isUnlimited} />
          <LimitRow
            label="Integrations per organization"
            value={limits?.maxIntegrationsPerOrg}
            unlimited={isUnlimited}
          />
          <LimitRow label="Events per month" value={limits?.maxEventsPerMonth} unlimited={isUnlimited} />
          <LimitRow label="Retention window" value={limits?.retentionDays} unlimited={isUnlimited} suffix=" days" />
        </div>
      </Fieldset>
    </div>
  );
}

function LimitRow({
  label,
  value,
  unlimited,
  suffix = "",
}: {
  label: string;
  value: number | undefined;
  unlimited: boolean;
  suffix?: string;
}) {
  return (
    <>
      <span className="text-gray-500 dark:text-gray-400">{label}</span>
      <span className="text-gray-900 dark:text-white font-medium text-right">
        {unlimited ? "Unlimited" : `${(value ?? 0).toLocaleString()}${suffix}`}
      </span>
    </>
  );
}
