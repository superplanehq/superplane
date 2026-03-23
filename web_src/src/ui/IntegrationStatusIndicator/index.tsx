import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { AlertTriangle, Check, ChevronDown, ChevronRight } from "lucide-react";
import { useCallback, useState } from "react";
import type { IntegrationsIntegrationDefinition } from "@/api-client";

export interface MissingIntegration {
  /** Integration type name, e.g. "github" */
  integrationName: string;
  /** Number of canvas nodes that need this integration */
  affectedNodeCount: number;
  /** The catalog definition (for icon slug, config fields, etc.) */
  definition?: IntegrationsIntegrationDefinition;
  /** Whether this integration was just connected (for success animation) */
  justConnected?: boolean;
}

export interface IntegrationStatusIndicatorProps {
  missingIntegrations: MissingIntegration[];
  onConnect: (integrationName: string) => void;
  readOnly?: boolean;
  canCreateIntegrations?: boolean;
}

export function IntegrationStatusIndicator({
  missingIntegrations,
  onConnect,
  readOnly = false,
  canCreateIntegrations = true,
}: IntegrationStatusIndicatorProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

  const handleToggle = useCallback(() => {
    setIsCollapsed((prev) => !prev);
  }, []);

  const handleConnect = useCallback(
    (integrationName: string) => {
      onConnect(integrationName);
    },
    [onConnect],
  );

  const activeCount = missingIntegrations.filter((m) => !m.justConnected).length;

  if (activeCount === 0) {
    return null;
  }

  if (isCollapsed) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            onClick={handleToggle}
            className="flex items-center gap-1.5 px-2.5 py-1.5 h-8 rounded-md bg-orange-100 dark:bg-orange-950/40 border border-orange-200 dark:border-orange-800 text-orange-800 dark:text-orange-200 hover:bg-orange-200 dark:hover:bg-orange-950/60 transition-colors text-xs font-medium cursor-pointer"
          >
            <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0" />
            <span>
              {activeCount} {activeCount === 1 ? "integration" : "integrations"}
            </span>
            <ChevronRight className="h-3 w-3" />
          </button>
        </TooltipTrigger>
        <TooltipContent>Show integrations that need setup</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <div className="w-80 bg-white dark:bg-gray-900 border border-orange-300 dark:border-orange-800 rounded-lg shadow-lg overflow-hidden animate-in fade-in slide-in-from-bottom-2 duration-300">
      <button
        onClick={handleToggle}
        className="w-full px-3 py-2 flex items-center justify-between border-b border-orange-200 dark:border-orange-900/50 bg-orange-50 dark:bg-orange-950/30 cursor-pointer hover:bg-orange-100 dark:hover:bg-orange-950/50 transition-colors"
      >
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-3.5 w-3.5 text-orange-600 dark:text-orange-400" />
          <p className="text-xs font-semibold text-orange-800 dark:text-orange-200">
            {activeCount} {activeCount === 1 ? "integration needs" : "integrations need"} setup
          </p>
        </div>
        <ChevronDown className="h-3.5 w-3.5 text-orange-600 dark:text-orange-400" />
      </button>
      <div className="max-h-64 overflow-y-auto">
        {missingIntegrations.map((integration) => {
          const displayName =
            getIntegrationTypeDisplayName(undefined, integration.integrationName) || integration.integrationName;

          return (
            <div
              key={integration.integrationName}
              className={`flex items-center gap-3 px-3 py-2.5 border-b border-gray-100 dark:border-gray-800 last:border-b-0 transition-opacity duration-300 ${
                integration.justConnected ? "opacity-50" : ""
              }`}
            >
              <IntegrationIcon
                integrationName={integration.integrationName}
                iconSlug={integration.definition?.icon}
                className="h-5 w-5 flex-shrink-0"
              />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{displayName}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {integration.affectedNodeCount} {integration.affectedNodeCount === 1 ? "node" : "nodes"}
                </p>
              </div>
              {integration.justConnected ? (
                <span className="flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400 font-medium">
                  <Check className="h-3.5 w-3.5" />
                  Connected
                </span>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  className="flex-shrink-0 h-7 text-xs"
                  onClick={() => handleConnect(integration.integrationName)}
                  disabled={readOnly || !canCreateIntegrations}
                >
                  Connect
                </Button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
