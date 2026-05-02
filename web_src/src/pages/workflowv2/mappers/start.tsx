import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type {
  TriggerRenderer,
  CustomFieldRendererContext,
  CustomFieldRenderer,
  NodeInfo,
  TriggerRendererContext,
  TriggerEventContext,
} from "./types";
import type { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { renderTimeAgo } from "@/components/TimeAgo";
import React from "react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    return { title: `Event received at ${new Date(context.event?.createdAt || "").toLocaleString()}`, subtitle: "" };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return flattenObject(context.event?.data || {});
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const nodeId = node.id;

    const customField = (
      onRunBase?: () => void,
      runContext?: { runDisabled?: boolean; runDisabledTooltip?: string },
    ) => {
      const onRun = onRunBase
        ? (initialData?: string) => {
            (window as any).__pendingRunData = { nodeId, initialData };
            onRunBase();
            setTimeout(() => {
              delete (window as any).__pendingRunData;
            }, 100);
          }
        : undefined;

      return startCustomFieldRenderer.render(node, {
        onRun,
        runDisabled: runContext?.runDisabled,
        runDisabledTooltip: runContext?.runDisabledTooltip,
      });
    };

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSlug: definition.icon || "play",
      iconColor: getColorClass("purple"),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: [],
      customField: customField,
      customFieldPosition: "before",
    };

    if (lastEvent) {
      props.lastEventData = {
        title: "Event emitted by trigger",
        subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
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
  render: (node: NodeInfo, context?: CustomFieldRendererContext): React.ReactNode => {
    const config = node.configuration as StartConfiguration;
    const templates = config?.templates || [];

    if (templates.length === 0) {
      return null;
    }

    const handleRun = (template: StartTemplate) => {
      if (context?.onRun && !context.runDisabled) {
        const payloadString = JSON.stringify(template.payload, null, 2);
        context.onRun(payloadString);
      }
    };

    const renderRunButton = (template: StartTemplate) => {
      if (!context?.onRun) {
        return null;
      }

      const button = (
        <Button
          size="sm"
          data-testid="start-template-run"
          disabled={context.runDisabled}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            handleRun(template);
          }}
          className="flex-shrink-0 h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
        >
          Run
        </Button>
      );

      if (!context.runDisabled || !context.runDisabledTooltip) {
        return button;
      }

      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <div aria-label={context.runDisabledTooltip} className="inline-flex flex-shrink-0">
              {button}
            </div>
          </TooltipTrigger>
          <TooltipContent>{context.runDisabledTooltip}</TooltipContent>
        </Tooltip>
      );
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
            {renderRunButton(template)}
          </div>
        ))}
      </div>
    );
  },
};
