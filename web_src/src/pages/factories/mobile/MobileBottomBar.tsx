import { cn } from "@/lib/utils";
import { Gauge, LayoutGrid, MoreHorizontal } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { Link, useLocation } from "react-router";

import {
  factoryHomePath,
  factorySettingsPath,
  factoryMorePath,
  factoryVelocityPath,
  firstFactoryLineId,
  workOrderBoardLineIdFromSearch,
} from "../lib/factoryPagePaths";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { MOBILE_BOTTOM_BAR_COPY } from "./mobileCopy";

const TAB_CLASSNAME =
  "flex h-full min-w-0 flex-1 flex-col items-center justify-center gap-1 text-xs font-medium tracking-[-0.01em] text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

function isActivePath(pathname: string, target: string): boolean {
  return pathname === target || pathname.startsWith(`${target}/`);
}

function lineIdFromPathname(pathname: string): string | undefined {
  const match = /\/lines\/([^/]+)/.exec(pathname);
  const lineId = match?.[1];
  if (!lineId || lineId === "new") {
    return undefined;
  }
  return decodeURIComponent(lineId);
}

function knownLineId(
  factory: { lines?: Array<{ id?: string }> | null } | null | undefined,
  lineId: string | null | undefined,
): string | undefined {
  if (!lineId || !factory?.lines?.some((line) => line.id === lineId)) {
    return undefined;
  }
  return lineId;
}

/**
 * Line the Board tab should open. Prefer the line in the current URL, then
 * the last line this shell showed, then the first line in the workspace.
 */
function useBoardLineId(factory: { lines?: Array<{ id?: string }> | null } | null | undefined): string | undefined {
  const { pathname, search } = useLocation();
  const routeLineId = knownLineId(factory, lineIdFromPathname(pathname) ?? workOrderBoardLineIdFromSearch(search));
  const [rememberedLineId, setRememberedLineId] = useState<string | undefined>();

  useEffect(() => {
    if (routeLineId) {
      setRememberedLineId(routeLineId);
    }
  }, [routeLineId]);

  return routeLineId ?? knownLineId(factory, rememberedLineId) ?? firstFactoryLineId(factory);
}

/** Phone navigation keeps account and workspace controls in More. */
export function MobileBottomBar() {
  const { organizationId, routeSegment, factory } = useFactoriesLayout();
  const { pathname } = useLocation();

  const boardLineId = useBoardLineId(factory);
  const boardHref = factoryHomePath(organizationId, routeSegment, boardLineId);
  const velocityHref = factoryVelocityPath(organizationId, routeSegment);
  const settingsHref = factorySettingsPath(organizationId, routeSegment);
  const moreHref = factoryMorePath(organizationId, routeSegment, boardLineId);
  const boardActive = pathname.includes("/lines/") || pathname.includes("/task/");

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
