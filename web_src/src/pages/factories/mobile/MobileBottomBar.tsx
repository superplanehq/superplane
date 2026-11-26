import { cn } from "@/lib/utils";
import { Gauge, LayoutGrid, MoreHorizontal } from "lucide-react";
import type { ReactNode } from "react";
import { Link, useLocation } from "react-router";

import {
  factoryHomePath,
  factorySettingsPath,
  factoryMorePath,
  factoryVelocityPath,
} from "../lib/factoryPagePaths";
import { isBoardPath } from "../layout/factoriesRail";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { MOBILE_BOTTOM_BAR_COPY } from "./mobileCopy";

const TAB_CLASSNAME =
  "flex h-full min-w-0 flex-1 flex-col items-center justify-center gap-1 text-xs font-medium tracking-[-0.01em] text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

function isActivePath(pathname: string, target: string): boolean {
  return pathname === target || pathname.startsWith(`${target}/`);
}

/** Phone navigation keeps account and workspace controls in More. */
export function MobileBottomBar() {
  const { organizationId, routeSegment } = useFactoriesLayout();
  const { pathname } = useLocation();

  const boardHref = factoryHomePath(organizationId, routeSegment);
  const velocityHref = factoryVelocityPath(organizationId, routeSegment);
  const settingsHref = factorySettingsPath(organizationId, routeSegment);
  const moreHref = factoryMorePath(organizationId, routeSegment);
  const boardActive = isBoardPath(pathname) || pathname.includes("/task/");

  return (
    <nav
      aria-label="Workspace"
      data-testid="mobile-bottom-bar"
      className="flex h-[calc(4rem+env(safe-area-inset-bottom))] shrink-0 items-stretch border-t border-border bg-background pb-[env(safe-area-inset-bottom)]"
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
      <BottomTab
        href={moreHref}
        label={MOBILE_BOTTOM_BAR_COPY.more}
        active={
          isActivePath(pathname, factoryMorePath(organizationId, routeSegment)) || isActivePath(pathname, settingsHref)
        }
        testId="mobile-tab-more"
      >
        <MoreHorizontal className="size-5" aria-hidden />
      </BottomTab>
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
