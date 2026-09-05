import { TriangleAlert } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { integrationDetailPath } from "@/lib/integrationSettingsPaths";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

import type { BrokenIntegration } from "./lib/brokenIntegrations";

interface BrokenIntegrationsBannerProps {
  integrations: BrokenIntegration[];
  /** Base path for the organization integrations settings pages. */
  integrationsBasePath: string;
  /** Whether the signed-in user can open Configure (`integrations:update`). Defaults to `true`. */
  canManageIntegrations?: boolean;
  className?: string;
}

/**
 * Amber banner listing organization integrations that need attention: an
 * expired key, an uninstalled app, or a setup left unfinished. Runs that
 * depend on these integrations fail until someone completes the repair
 * step named on each row.
 */
export function BrokenIntegrationsBanner({
  integrations,
  integrationsBasePath,
  canManageIntegrations = true,
  className,
}: BrokenIntegrationsBannerProps) {
  if (integrations.length === 0) {
    return null;
  }

  const title =
    integrations.length === 1 ? "1 integration needs attention" : `${integrations.length} integrations need attention`;

  return (
    <div
      role="status"
      data-testid="broken-integrations-banner"
      className={cn(
        "flex w-full flex-col gap-2 rounded-lg border border-amber-200 bg-amber-50/90 px-3.5 py-2.5 text-sm text-amber-950",
        "dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-100",
        className,
      )}
    >
      <div className="flex items-center gap-2">
        <TriangleAlert className="h-4 w-4 shrink-0 text-amber-700 dark:text-amber-300" aria-hidden />
        <span className="font-medium">{title}</span>
      </div>
      <ul className="flex flex-col gap-1.5">
        {integrations.map((integration) => {
          const displayName = getIntegrationTypeDisplayName(undefined, integration.integrationName);
          return (
            <li
              key={integration.id}
              data-testid="broken-integration-row"
              className="flex flex-wrap items-center justify-between gap-2 rounded-md bg-white/60 px-2.5 py-1.5 dark:bg-black/10"
            >
              <div className="flex min-w-0 items-center gap-2">
                <IntegrationIcon integrationName={integration.integrationName} className="h-4 w-4 shrink-0" />
                <div className="min-w-0">
                  <p className="truncate font-medium leading-tight">{displayName || integration.integrationName}</p>
                  {integration.description ? (
                    <p className="truncate text-xs text-amber-800/80 dark:text-amber-200/70">
                      {integration.description}
                    </p>
                  ) : null}
                </div>
              </div>
              {canManageIntegrations ? (
                <Button
                  asChild
                  variant="outline"
                  size="sm"
                  className="shrink-0 border-amber-300 bg-amber-100/70 text-amber-950 hover:bg-amber-100 dark:border-amber-600 dark:bg-amber-900/40 dark:text-amber-50 dark:hover:bg-amber-900/60"
                >
                  <Link to={integrationDetailPath(integrationsBasePath, integration.id)}>
                    {integration.actionLabel}
                  </Link>
                </Button>
              ) : null}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
