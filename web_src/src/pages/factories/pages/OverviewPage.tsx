import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Badge } from "@/components/ui/badge";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { ChevronRight, Loader2 } from "lucide-react";
import { Link } from "react-router-dom";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryLineDetailPath, workOrderDetailPath, workOrdersPath, automationsPath } from "../lib/factoryPagePaths";
import { getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { factoryContentBodyClassName, factoryContentHeaderClassName } from "./factoryPageLayoutStyles";

const MAX_ROWS = 8;

export function OverviewPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const {
    data: workOrders = [],
    isLoading: workOrdersLoading,
    error: workOrdersError,
  } = useFactoryWorkOrders(organizationId, factoryId);

  const recentOrders = [...workOrders]
    .sort((a, b) => {
      const aTime = Date.parse(a.updatedAt ?? a.createdAt ?? "") || 0;
      const bTime = Date.parse(b.updatedAt ?? b.createdAt ?? "") || 0;
      return bTime - aTime;
    })
    .slice(0, MAX_ROWS);

  const lines = factory?.lines ?? [];

  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div>
          <Heading level={1} className="!text-xl text-gray-900 dark:text-gray-100">
            Overview
          </Heading>
          <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Your workspace at a glance. Content for this page comes next.
          </Text>
        </div>
      </header>

      <div className={factoryContentBodyClassName}>
        <div className="grid gap-6 lg:grid-cols-2">
          <WorkOrdersOverviewCard
            organizationId={organizationId}
            factoryId={factoryId}
            orders={recentOrders}
            isLoading={workOrdersLoading}
            error={workOrdersError}
          />
          <LinesOverviewCard organizationId={organizationId} factoryId={factoryId} lines={lines} />
        </div>
      </div>
    </>
  );
}

function WorkOrdersOverviewCard({
  organizationId,
  factoryId,
  orders,
  isLoading,
  error,
}: {
  organizationId: string;
  factoryId: string;
  orders: FactoriesWorkOrder[];
  isLoading: boolean;
  error: Error | null;
}) {
  return (
    <section
      className="overflow-hidden rounded-lg border border-slate-950/10 bg-white dark:border-gray-700/70 dark:bg-gray-900"
      data-testid="overview-work-orders-card"
    >
      <div className="flex items-center justify-between border-b border-slate-950/10 px-4 py-3 dark:border-gray-700/70">
        <div>
          <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Work Orders</h2>
          <p className="text-xs text-gray-500 dark:text-gray-400">Recent activity by status.</p>
        </div>
        <Link
          to={workOrdersPath(organizationId, factoryId)}
          className="text-xs font-medium text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-gray-100"
          data-testid="overview-work-orders-view-all"
        >
          View all
        </Link>
      </div>

      <div>
        {error ? (
          <p className="px-4 py-6 text-sm text-red-600 dark:text-red-400">Failed to load work orders.</p>
        ) : isLoading ? (
          <p className="px-4 py-6 text-sm text-gray-500 dark:text-gray-400">Loading work orders…</p>
        ) : orders.length === 0 ? (
          <p className="px-4 py-6 text-sm text-gray-500 dark:text-gray-400">No work orders yet.</p>
        ) : (
          <ul>
            {orders.map((order) => {
              const status = getWorkOrderDisplayStatus(order);
              const statusMeta = getWorkOrderDisplayStatusMeta(status);
              const href = order.id ? workOrderDetailPath(organizationId, factoryId, order.id) : "#";
              const updatedAt = order.updatedAt ?? order.createdAt;
              const timeLabel = updatedAt ? formatTimeAgo(new Date(updatedAt)) : "—";
              return (
                <li
                  key={order.id ?? order.title}
                  className="border-b border-slate-950/5 last:border-b-0 dark:border-gray-700/50"
                >
                  <Link
                    to={href}
                    className="group flex items-center gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/40"
                    data-testid={`overview-work-order-row-${order.id}`}
                  >
                    <Badge
                      variant="outline"
                      className={cn(
                        "inline-flex shrink-0 items-center px-2 py-0.5 text-[11px] font-medium",
                        statusMeta.className,
                      )}
                    >
                      {status === "running" ? (
                        <Loader2 className="mr-1 inline h-3 w-3 animate-spin" aria-hidden />
                      ) : null}
                      {statusMeta.label}
                    </Badge>
                    <p className="min-w-0 flex-1 truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                      {order.title || "Untitled work order"}
                    </p>
                    <span className="shrink-0 text-xs text-gray-500 dark:text-gray-400">{timeLabel}</span>
                    <ChevronRight
                      className="h-4 w-4 shrink-0 text-gray-400 transition group-hover:text-gray-700 dark:text-gray-500 dark:group-hover:text-gray-300"
                      aria-hidden
                    />
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}

function LinesOverviewCard({
  organizationId,
  factoryId,
  lines,
}: {
  organizationId: string;
  factoryId: string;
  lines: FactoriesFactoryLine[];
}) {
  return (
    <section
      className="overflow-hidden rounded-lg border border-slate-950/10 bg-white dark:border-gray-700/70 dark:bg-gray-900"
      data-testid="overview-lines-card"
    >
      <div className="flex items-center justify-between border-b border-slate-950/10 px-4 py-3 dark:border-gray-700/70">
        <div>
          <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Automations</h2>
          <p className="text-xs text-gray-500 dark:text-gray-400">Lines and their steps.</p>
        </div>
        <Link
          to={automationsPath(organizationId, factoryId)}
          className="text-xs font-medium text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-gray-100"
          data-testid="overview-lines-view-all"
        >
          View all
        </Link>
      </div>

      {lines.length === 0 ? (
        <p className="px-4 py-6 text-sm text-gray-500 dark:text-gray-400">No lines configured yet.</p>
      ) : (
        <ul>
          {lines.map((line) => {
            const stepsCount = line.steps?.length ?? 0;
            const href = line.id ? factoryLineDetailPath(organizationId, factoryId, line.id) : "#";
            return (
              <li
                key={line.id ?? line.name}
                className="border-b border-slate-950/5 last:border-b-0 dark:border-gray-700/50"
              >
                <Link
                  to={href}
                  className="group flex items-center gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/40"
                  data-testid={`overview-line-row-${line.id}`}
                >
                  <p className="min-w-0 flex-1 truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                    {line.name || "Unnamed line"}
                  </p>
                  <span className="shrink-0 text-xs text-gray-500 dark:text-gray-400">
                    {stepsCount} {stepsCount === 1 ? "phase" : "phases"}
                  </span>
                  <ChevronRight
                    className="h-4 w-4 shrink-0 text-gray-400 transition group-hover:text-gray-700 dark:text-gray-500 dark:group-hover:text-gray-300"
                    aria-hidden
                  />
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
