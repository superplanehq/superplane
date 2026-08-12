import type { FactoriesWorkOrder } from "@/api-client";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import { NavLink } from "react-router-dom";
import { workOrderDetailPath } from "../lib/factoryPagePaths";
import { getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import type { OnboardingNavProgress } from "../pages/onboarding/onboardingStorybookContextValue";
import { useOnboardingStorybook } from "../pages/onboarding/useOnboardingStorybook";
import { FACTORIES_NAV_ITEMS, type FactoriesNavKind } from "./factoriesNavItems";

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

type NavItemState = "unlocked" | "analyzing" | "locked";

function navItemState(
  id: FactoriesNavKind,
  progressive: boolean,
  progress: OnboardingNavProgress | undefined,
): NavItemState {
  if (!progressive || !progress) return "unlocked";
  if (id === "overview") return "unlocked";

  if (id === "wiki" || id === "velocity") {
    if (progress.analyzingRepo) return "analyzing";
    if (progress.repoReady) return "unlocked";
    return "locked";
  }
  if (id === "work-orders") {
    if (progress.analyzingIssues) return "analyzing";
    if (progress.issuesReady) return "unlocked";
    return "locked";
  }
  if (id === "lines" || id === "automations") {
    if (progress.analyzingAgent) return "analyzing";
    if (progress.agentReady) return "unlocked";
    return "locked";
  }
  return "locked";
}

function NavSetupSpinner() {
  return <Loader2 className="ml-auto size-3.5 shrink-0 animate-spin text-muted-foreground" aria-hidden />;
}

export function FactoriesNav({ organizationId, factoryId, recentWorkOrders }: FactoriesNavProps) {
  const onboarding = useOnboardingStorybook();
  const progressive = Boolean(onboarding?.pending);
  const progress = onboarding?.setupProgress;
  // No recent work orders during setup — the workspace has none yet.
  const showRecent = !progressive;

  return (
    <nav className="flex flex-1 flex-col gap-4 px-2 pt-2 pb-4" data-testid="factories-nav">
      <ul className="flex flex-col gap-0.5">
        {FACTORIES_NAV_ITEMS.map((item) => {
          const Icon = item.Icon;
          const href = item.buildHref(organizationId, factoryId);
          const state = navItemState(item.id, progressive, progress);

          if (state === "unlocked") {
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
          }

          return (
            <li key={item.id}>
              <div
                aria-disabled="true"
                data-testid={`factories-nav-${item.id}`}
                data-state={state}
                className="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-[13px] tracking-[-0.01em] text-muted-foreground/50"
                title={
                  state === "analyzing"
                    ? "Generating while you continue setup…"
                    : "Complete the matching setup step to unlock"
                }
              >
                <Icon className="size-[15px] shrink-0 opacity-50" strokeWidth={1.75} aria-hidden />
                <span>{item.label}</span>
                {state === "analyzing" ? <NavSetupSpinner /> : null}
              </div>
            </li>
          );
        })}
      </ul>

      {showRecent && recentWorkOrders.length > 0 ? (
        <section aria-label="Recent work orders">
          <p className="px-2.5 pb-2 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">
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
