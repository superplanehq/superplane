import { cn } from "@/lib/utils";
import { ChartLine, Home } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link, useLocation } from "react-router";
import { factoryHomePath, factoryVelocityPath } from "../lib/factoryPagePaths";
import { factoriesRailControlClassName, isBoardPath, isVelocityPath } from "./factoriesRail";

interface FactoriesSidebarNavProps {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
}

function railLinkClassName(isCurrent: boolean) {
  return cn(factoriesRailControlClassName, isCurrent && "bg-sidebar-accent text-foreground");
}

function RailNavLink({
  to,
  label,
  Icon,
  testId,
  isCurrent,
}: {
  to: string;
  label: string;
  Icon: LucideIcon;
  testId: string;
  isCurrent: boolean;
}) {
  return (
    <Link
      to={to}
      aria-label={label}
      title={label}
      aria-current={isCurrent ? "page" : undefined}
      data-testid={testId}
      className={railLinkClassName(isCurrent)}
    >
      <Icon className="size-3.5" aria-hidden />
    </Link>
  );
}

/**
 * Icon rail under the workspace switcher: the line board and Velocity.
 * Workspace settings open from the workspace switcher.
 */
export function FactoriesSidebarNav({ organizationId, factoryKey, lineId }: FactoriesSidebarNavProps) {
  const { pathname } = useLocation();
  const boardHref = factoryHomePath(organizationId, factoryKey, lineId);
  const velocityHref = factoryVelocityPath(organizationId, factoryKey);

  return (
    <nav className="flex flex-col items-center gap-1 px-1.5" aria-label="Workspace" data-testid="factories-sidebar-nav">
      <RailNavLink
        to={boardHref}
        label="Board"
        Icon={Home}
        testId="factories-nav-board"
        isCurrent={isBoardPath(pathname)}
      />
      <RailNavLink
        to={velocityHref}
        label="Velocity"
        Icon={ChartLine}
        testId="factories-nav-velocity"
        isCurrent={isVelocityPath(pathname)}
      />
    </nav>
  );
}
