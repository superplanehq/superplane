import { canvasesInvokeNodeTriggerAction } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { showErrorToast } from "@/utils/toast";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { TriggerRenderer, CustomFieldRenderer, NodeInfo, TriggerRendererContext, TriggerEventContext } from "./types";
import { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import React from "react";
import { Play } from "lucide-react";

interface StartTemplate {
  name: string;
  payload: Record<string, unknown>;
}

interface StartConfiguration {
  templates?: StartTemplate[];
}

/**
 * Default renderer for the start trigger
 */
export const startTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    return { title: `Event received at ${new Date(context.event?.createdAt || "").toLocaleString()}`, subtitle: "" };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return flattenObject(context.event?.data || {});
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent, canvasId, organizationId, runDisabled, runDisabledTooltip } = context;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSlug: definition.icon || "play",
      iconColor: getColorClass("purple"),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: [],
      customField: () =>
        startCustomFieldRenderer.render(node, {
          canvasId,
          organizationId,
          runDisabled,
          runDisabledTooltip,
        }),
      customFieldPosition: "before",
    };

    if (lastEvent) {
      props.lastEventData = {
        title: "Event emitted by trigger",
        subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

/**
 * Custom field renderer for the start trigger that displays templates with Run buttons
 * This is only used internally by startTriggerRenderer, not registered in the global registry
 */
const startCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo, context): React.ReactNode => {
    const config = node.configuration as StartConfiguration;
    const templates = config?.templates || [];

    if (templates.length === 0) {
      return null;
    }

    const handleRun = async (templateName: string) => {
      if (!context?.canvasId || context.runDisabled) {
        return;
      }

      try {
        await canvasesInvokeNodeTriggerAction(
          withOrganizationHeader({
            organizationId: context.organizationId,
            path: {
              canvasId: context.canvasId,
              nodeId: node.id,
              actionName: "run",
            },
            body: {
              parameters: {
                templateName,
              },
            },
          }),
        );
      } catch (_error) {
        showErrorToast("Failed to start run");
      }
    };

    return (
      <div className="px-2 py-1.5 flex flex-col gap-1.5">
        {templates.map((template, index) => (
          <div key={index} className="flex items-center justify-between min-w-0">
            <div className="flex items-center min-w-0 flex-1">
              <div className="w-4 h-4 mr-2 flex-shrink-0">
                <Play size={16} className="text-gray-500" />
              </div>
              <span className="text-[13px] font-medium font-inter text-gray-500 truncate">{template.name}</span>
            </div>
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="inline-flex">
                  <Button
                    size="sm"
                    data-testid="start-template-run"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      void handleRun(template.name);
                    }}
                    disabled={!context?.canvasId || context.runDisabled}
                    className="flex-shrink-0 h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
                  >
                    Run
                  </Button>
                </div>
              </TooltipTrigger>
              {context?.runDisabled && context.runDisabledTooltip ? (
                <TooltipContent side="top">{context.runDisabledTooltip}</TooltipContent>
              ) : null}
            </Tooltip>
          </div>
        ))}
      </div>
    );
  },
};
