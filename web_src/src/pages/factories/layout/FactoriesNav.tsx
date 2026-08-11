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

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "flex h-8 items-center gap-2.5 rounded-md px-2.5 text-[13px] tracking-[-0.01em] transition-colors",
    isActive
      ? "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
      : "font-normal text-foreground/80 hover:bg-sidebar-accent/70 hover:text-foreground",
  );

export function FactoriesNav({ organizationId, factoryId, recentWorkOrders }: FactoriesNavProps) {
  return (
    <>
      <nav className="flex flex-col gap-0.5 px-2" data-testid="factories-nav" aria-label="Primary">
        {FACTORIES_NAV_ITEMS.map((item) => {
          const Icon = item.Icon;
          const href = item.buildHref(organizationId, factoryId);
          return (
            <NavLink key={item.id} to={href} data-testid={`factories-nav-${item.id}`} className={navLinkClass}>
              <Icon className="size-[15px] shrink-0 opacity-80" strokeWidth={1.75} aria-hidden />
              <span className="truncate">{item.label}</span>
            </NavLink>
          );
        })}
      </nav>

      {recentWorkOrders.length > 0 ? (
        <div className="mt-5 px-2" aria-label="Recent work orders">
          <div className="px-2.5 pb-2 text-[11px] font-medium tracking-[0.04em] text-muted-foreground">Recent</div>
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
                        "block rounded-md px-2.5 py-1.5 transition-colors",
                        isActive ? "bg-sidebar-accent" : "hover:bg-sidebar-accent",
                      )
                    }
                    data-testid={`factories-nav-recent-${order.id}`}
                  >
                    <p className="truncate text-[13px] leading-snug tracking-[-0.01em] text-foreground">
                      {order.title || "Untitled work order"}
                    </p>
                    <p className="mt-0.5 flex items-center gap-1.5 text-[12px] text-muted-foreground">
                      <span className={cn("size-1.5 shrink-0 rounded-full", dotClass)} aria-hidden />
                      <span>{statusMeta.label}</span>
                    </p>
                  </NavLink>
                </li>
              );
            })}
          </ul>
        </div>
      ) : null}
    </>
  );
}
