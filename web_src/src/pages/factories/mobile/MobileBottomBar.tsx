import { useAccount } from "@/contexts/useAccount";
import { useOrganization } from "@/hooks/useOrganizationData";
import { cn } from "@/lib/utils";
import { Gauge, LayoutGrid, Plus, Settings } from "lucide-react";
import type { ReactNode } from "react";
import { Link, useLocation } from "react-router";

import { SidebarUserMenu } from "../layout/SidebarUserMenu";
import {
  factoryHomePath,
  factorySettingsWorkspaceGeneralPath,
  factoryVelocityPath,
  firstFactoryLineId,
} from "../lib/factoryPagePaths";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { MOBILE_BOTTOM_BAR_COPY } from "./mobileCopy";

const TAB_CLASSNAME =
  "flex h-full min-w-0 flex-1 flex-col items-center justify-center gap-0.5 text-[10px] font-medium tracking-[-0.01em] text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

function isActivePath(pathname: string, target: string): boolean {
  return pathname === target || pathname.startsWith(`${target}/`);
}

/**
 * Phone navigation. Create task sits in the middle because it is the main
 * action on every workspace page. The other slots open full pages.
 */
export function MobileBottomBar({ canCreateWorkOrder }: { canCreateWorkOrder: boolean }) {
  const { organizationId, routeSegment, factory, openCreateWorkOrder } = useFactoriesLayout();
  const { pathname } = useLocation();
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId);

  const boardHref = factoryHomePath(organizationId, routeSegment, firstFactoryLineId(factory));
  const velocityHref = factoryVelocityPath(organizationId, routeSegment);
  const settingsHref = factorySettingsWorkspaceGeneralPath(organizationId, routeSegment);
  const boardActive = pathname.includes("/lines/") || pathname.includes("/task/");

  return (
    <nav
      aria-label="Workspace"
      data-testid="mobile-bottom-bar"
      className="flex h-[calc(3.5rem+env(safe-area-inset-bottom))] shrink-0 items-stretch border-t border-border bg-background pb-[env(safe-area-inset-bottom)]"
    >
      <BottomTab href={boardHref} label={MOBILE_BOTTOM_BAR_COPY.board} active={boardActive} testId="mobile-tab-board">
        <LayoutGrid className="size-5" aria-hidden />
      </BottomTab>
      <BottomTab
        href={velocityHref}
        label={MOBILE_BOTTOM_BAR_COPY.velocity}
        active={isActivePath(pathname, velocityHref)}
        testId="mobile-tab-velocity"
      >
        <Gauge className="size-5" aria-hidden />
      </BottomTab>
      <div className="flex flex-1 items-center justify-center">
        <button
          type="button"
          onClick={openCreateWorkOrder}
          disabled={!canCreateWorkOrder}
          aria-label={MOBILE_BOTTOM_BAR_COPY.createTask}
          title={MOBILE_BOTTOM_BAR_COPY.createTask}
          data-testid="mobile-create-task"
          className="flex size-12 -translate-y-3 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-opacity hover:opacity-90 disabled:opacity-40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          <Plus className="size-6" aria-hidden />
        </button>
      </div>
      <BottomTab
        href={settingsHref}
        label={MOBILE_BOTTOM_BAR_COPY.settings}
        active={isActivePath(pathname, settingsHref)}
        testId="mobile-tab-settings"
      >
        <Settings className="size-5" aria-hidden />
      </BottomTab>
      <div className="flex min-w-0 flex-1 items-center justify-center [&>div]:border-t-0 [&>div]:py-0">
        <SidebarUserMenu
          organizationId={organizationId}
          factoryKey={routeSegment}
          userName={account?.name ?? "You"}
          userAvatarUrl={account?.avatar_url}
          organizationName={organization?.metadata?.name || "Organization"}
        />
      </div>
    </nav>
  );
}

function BottomTab({
  href,
  label,
  active,
  testId,
  children,
}: {
  href: string;
  label: string;
  active: boolean;
  testId: string;
  children: ReactNode;
}) {
  return (
    <Link
      to={href}
      aria-current={active ? "page" : undefined}
      data-testid={testId}
      className={cn(TAB_CLASSNAME, active && "text-foreground")}
    >
      {children}
      <span className="truncate">{label}</span>
    </Link>
  );
}
