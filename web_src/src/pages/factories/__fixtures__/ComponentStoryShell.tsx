import { MemoryRouter } from "react-router";
import type { ReactNode } from "react";
import { TooltipProvider } from "@/ui/tooltip";

interface ComponentStoryShellProps {
  children: ReactNode;
  /** Initial URL for `MemoryRouter`. Defaults to `/`. */
  initialPath?: string;
  className?: string;
}

/**
 * Minimal wrapper for factory *component* stories that use `Link`
 * (`react-router`) or `PermissionTooltip` (needs tooltip context).
 * Full-page stories should use `FactoriesHarness` instead so hooks fetch.
 */
export function ComponentStoryShell({
  children,
  initialPath = "/",
  className = "min-h-screen bg-gray-50 p-6 dark:bg-gray-950",
}: ComponentStoryShellProps) {
  return (
    <MemoryRouter initialEntries={[initialPath]}>
      <TooltipProvider delayDuration={150}>
        <div className={className}>{children}</div>
      </TooltipProvider>
    </MemoryRouter>
  );
}
