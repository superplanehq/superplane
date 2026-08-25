import type { FactoriesWorkOrder } from "@/api-client";
import { cn } from "@/lib/utils";
import { NavLink } from "react-router";
import { factoryHomePath } from "../lib/factoryPagePaths";
import { getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { FACTORIES_NAV_ITEMS } from "./factoriesNavItems";

interface FactoriesNavProps {
  organizationId: string;
  factoryKey: string;
  recentWorkOrders: FactoriesWorkOrder[];
}

import type { WorkOrderDisplayStatus } from "../lib/workOrderProgress";

const RECENT_STATUS_DOT_CLASS: Record<WorkOrderDisplayStatus, string> = {
  draft: "bg-[color:var(--status-draft-dot)]",
  waiting: "bg-[color:var(--status-waiting-dot)]",
  running: "bg-[color:var(--status-running-dot)]",
  failed: "bg-[color:var(--status-failed-dot)]",
  rejected: "bg-[color:var(--status-failed-dot)]",
  completed: "bg-[color:var(--status-completed-dot)]",
  cancelled: "bg-[color:var(--status-cancelled-dot)]",
};

export function FactoriesNav({ organizationId, factoryKey, recentWorkOrders }: FactoriesNavProps) {
  return (
    <nav className="flex flex-1 flex-col gap-4 px-2 pt-2 pb-4" data-testid="factories-nav">
      <ul className="flex flex-col gap-0.5">
        {FACTORIES_NAV_ITEMS.map((item) => {
          const Icon = item.Icon;
          const href = item.buildHref(organizationId, factoryKey);

          return (
            <li key={item.id}>
              <NavLink
                to={href}
                data-testid={`factories-nav-${item.id}`}
                className={({ isActive }) =>
                  cn(
                    "group flex items-center gap-2 rounded-md px-2.5 py-1.5 text-[13px] tracking-[-0.01em] text-foreground/80 hover:bg-sidebar-accent hover:text-foreground",
                    isActive && "bg-sidebar-accent font-medium text-foreground",
                  )
                }
              >
                <Icon className="size-[15px] shrink-0 opacity-80" strokeWidth={1.75} aria-hidden />
                <span>{item.label}</span>
              </NavLink>
            </li>
          );
        })}
      </ul>

      {recentWorkOrders.length > 0 ? (
        <section aria-label="Recent work orders">
          <p className="px-2.5 pb-2 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">
            Recent
          </p>
          <ul className="flex flex-col gap-0.5">
            {recentWorkOrders.map((order) => {
              if (!order.id || order.number === undefined) {
                return null;
              }
              const status = getWorkOrderDisplayStatus(order);
              const statusMeta = getWorkOrderDisplayStatusMeta(status);
              const dotClass = RECENT_STATUS_DOT_CLASS[status];
              return (
                <li key={order.id}>
                  <NavLink
                    to={factoryHomePath(organizationId, factoryKey)}
                    className={({ isActive }) =>
                      cn(
                        "group block rounded-md px-2.5 py-1.5 text-[13px] tracking-[-0.01em] text-foreground/80 hover:bg-sidebar-accent hover:text-foreground",
                        isActive && "bg-sidebar-accent font-medium text-foreground",
                      )
                    }
                    data-testid={`factories-nav-recent-${order.id}`}
                  >
                    <p className="truncate">{order.title || "Untitled work order"}</p>
                    <p className="mt-0.5 inline-flex items-center gap-1.5 text-[11px] text-muted-foreground">
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
