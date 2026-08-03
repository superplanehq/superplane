import type { FactoriesFactory } from "@/api-client";
import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { SlidersHorizontal } from "lucide-react";

export type FactoryDetailTab = "my-work" | "work-orders" | "lines" | "apps";

interface FactoryDetailHeaderProps {
  factory: FactoriesFactory;
  backHref: string;
  activeTab: FactoryDetailTab;
  onTabChange: (tab: FactoryDetailTab) => void;
  myWorkCount: number;
  workOrdersCount: number;
  linesCount: number;
  appsCount: number;
  needsAttentionCount: number;
  canCreate: boolean;
  permissionsLoading: boolean;
  onCreateClick: () => void;
}

export function FactoryDetailHeader({
  factory,
  backHref,
  activeTab,
  onTabChange,
  myWorkCount,
  workOrdersCount,
  linesCount,
  appsCount,
  needsAttentionCount,
  canCreate,
  permissionsLoading,
  onCreateClick,
}: FactoryDetailHeaderProps) {
  return (
    <div className="border-b border-slate-200 bg-white px-6 py-6 dark:border-gray-700/70 dark:bg-gray-900 sm:px-8">
      <div className="mx-auto max-w-5xl">
        <div className="mb-4 flex flex-wrap items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <Link href={backHref} className="hover:text-slate-900 dark:hover:text-gray-100">
            Software Factory
          </Link>
          <span aria-hidden>·</span>
          <span className="inline-flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-emerald-500" aria-hidden />
            Operational
          </span>
        </div>

        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0 flex-1">
            <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">{factory.name}</h1>
            <p className="mt-2 max-w-3xl text-sm text-gray-500 dark:text-gray-400">
              {factory.description ||
                "Plans, implements, verifies, and delivers well-specified product work for this factory."}
            </p>
          </div>

          <div className="flex items-center gap-2">
            <Button type="button" variant="outline" size="icon-sm" disabled aria-label="Factory settings (coming soon)">
              <SlidersHorizontal className="h-4 w-4" />
            </Button>
            <PermissionTooltip
              allowed={canCreate || permissionsLoading}
              message="You don't have permission to create work orders."
            >
              <Button
                type="button"
                onClick={onCreateClick}
                disabled={!canCreate}
                className={cn(appDarkModeClasses.primaryAction)}
                data-testid="work-order-list-create-button"
              >
                <Icon name="plus" />
                New work
              </Button>
            </PermissionTooltip>
          </div>
        </div>

        <Tabs
          value={activeTab}
          onValueChange={(value) => onTabChange(value as FactoryDetailTab)}
          className="mt-6 gap-0"
        >
          <TabsList className="h-8 bg-slate-100 dark:bg-gray-800">
            <TabsTrigger value="my-work" data-testid="factory-tab-my-work">
              My work
              {myWorkCount > 0 ? <TabCountBadge count={myWorkCount} highlight={needsAttentionCount > 0} /> : null}
            </TabsTrigger>
            <TabsTrigger value="work-orders" data-testid="factory-tab-work-orders">
              Work orders
              {workOrdersCount > 0 ? <TabCountBadge count={workOrdersCount} /> : null}
            </TabsTrigger>
            <TabsTrigger value="lines" data-testid="factory-tab-lines">
              Lines
              {linesCount > 0 ? <TabCountBadge count={linesCount} /> : null}
            </TabsTrigger>
            <TabsTrigger value="apps" data-testid="factory-tab-apps">
              Apps
              {appsCount > 0 ? <TabCountBadge count={appsCount} /> : null}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
    </div>
  );
}

function TabCountBadge({ count, highlight = false }: { count: number; highlight?: boolean }) {
  return (
    <span
      className={cn(
        "ml-1 inline-flex min-w-5 items-center justify-center rounded-full px-1.5 py-0.5 text-[10px] font-semibold leading-none",
        highlight
          ? "bg-amber-500/20 text-amber-700 dark:bg-amber-500/25 dark:text-amber-200"
          : "bg-slate-200/80 text-slate-600 dark:bg-gray-700 dark:text-gray-300",
      )}
    >
      {count}
    </span>
  );
}
