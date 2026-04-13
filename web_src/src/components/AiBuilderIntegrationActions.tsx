import { Button } from "@/components/ui/button";
import type { AiIntegrationAction } from "@/ui/BuildingBlocksSidebar/agentChat";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Check, Plus } from "lucide-react";

export type AiBuilderIntegrationActionsProps = {
  actions: AiIntegrationAction[];
  onConnect: (integrationName: string) => void;
  disabled: boolean;
  connectedIntegrationNames?: Set<string>;
};

export function AiBuilderIntegrationActions({
  actions,
  onConnect,
  disabled,
  connectedIntegrationNames,
}: AiBuilderIntegrationActionsProps) {
  if (actions.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-1.5 pt-2">
      {actions.map((action) => {
        const isConnected = connectedIntegrationNames?.has(action.integrationName.toLowerCase()) ?? false;

        if (isConnected) {
          return (
            <span
              key={action.integrationName}
              className="inline-flex items-center gap-1.5 rounded-md border border-green-200 bg-green-50 py-1 px-2.5 text-xs font-normal text-green-700"
            >
              <IntegrationIcon integrationName={action.integrationName} className="h-3.5 w-3.5" />
              <Check className="h-3 w-3" />
              {action.label.replace(/^Connect\s+/i, "")} connected
            </span>
          );
        }

        return (
          <Button
            key={action.integrationName}
            variant="outline"
            size="sm"
            disabled={disabled}
            onClick={() => onConnect(action.integrationName)}
            className="h-auto py-1 px-2.5 text-xs font-normal gap-1.5"
          >
            <IntegrationIcon integrationName={action.integrationName} className="h-3.5 w-3.5" />
            <Plus className="h-3 w-3" />
            {action.label}
          </Button>
        );
      })}
    </div>
  );
}
