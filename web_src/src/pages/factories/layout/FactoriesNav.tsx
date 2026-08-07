import type { FactoriesWorkOrder } from "@/api-client";
import { cn } from "@/lib/utils";
import { NavLink } from "react-router-dom";
import { workOrderDetailPath } from "../lib/factoryPagePaths";
import { getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { FACTORIES_NAV_ITEMS } from "./factoriesNavItems";

interface FactoriesNavProps {
  organizationId: string;
  factoryId: string;
  recentWorkOrders: FactoriesWorkOrder[];
}

const RECENT_STATUS_DOT_CLASS: Record<string, string> = {
  draft: "bg-gray-400",
  open: "bg-sky-500",
  running: "bg-violet-500",
  failed: "bg-red-500",
  completed: "bg-emerald-500",
  rejected: "bg-gray-400",
  closedFailed: "bg-red-500",
};

export function FactoriesNav({ organizationId, factoryId, recentWorkOrders }: FactoriesNavProps) {
  return (
    <nav className="flex flex-1 flex-col gap-6 px-2 pt-4 pb-4" data-testid="factories-nav">
      <ul className="flex flex-col gap-0.5">
        {FACTORIES_NAV_ITEMS.map((item) => {
          const Icon = item.Icon;
          const href = item.buildHref(organizationId, factoryId);
          return (
            <li key={item.id}>
              <NavLink
                to={href}
                data-testid={`factories-nav-${item.id}`}
                className={({ isActive }) =>
                  cn(
                    "group flex items-center gap-2 rounded-md px-2 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-800 dark:hover:text-gray-100",
                    isActive && "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100",
                  )
                }
              >
                <Icon className="h-4 w-4 shrink-0" aria-hidden />
                <span>{item.label}</span>
              </NavLink>
            </li>
          );
        })}
      </ul>

      {recentWorkOrders.length > 0 ? (
        <section aria-label="Recent work orders">
          <p className="px-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-500">
            Recent
          </p>
          <ul className="flex flex-col gap-0.5">
            {recentWorkOrders.map((order) => {
              if (!order.id) {
                return null;
              }
              const status = getWorkOrderDisplayStatus(order);
              const statusMeta = getWorkOrderDisplayStatusMeta(status);
              const dotClass = RECENT_STATUS_DOT_CLASS[status] ?? "bg-gray-400";
              return (
                <li key={order.id}>
                  <NavLink
                    to={workOrderDetailPath(organizationId, factoryId, order.id)}
                    className={({ isActive }) =>
                      cn(
                        "group block rounded-md px-2 py-1.5 text-sm text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-800 dark:hover:text-gray-100",
                        isActive && "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100",
                      )
                    }
                    data-testid={`factories-nav-recent-${order.id}`}
                  >
                    <p className="truncate font-medium">{order.title || "Untitled work order"}</p>
                    <p className="mt-0.5 inline-flex items-center gap-1.5 text-[11px] text-gray-500 dark:text-gray-400">
                      <span className={cn("h-1.5 w-1.5 rounded-full", dotClass)} aria-hidden />
                      {statusMeta.label}
                    </p>
                  </NavLink>
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}
    </nav>
  );
}
