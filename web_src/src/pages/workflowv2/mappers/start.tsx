import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer, CustomFieldRenderer, NodeInfo, TriggerRendererContext, TriggerEventContext } from "./types";
import { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import React from "react";
import { Button } from "@/components/ui/button";
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
    const { node, definition, lastEvent } = context;
    const nodeId = node.id;

    // Create customField as a function that will receive onRun when ComponentBase renders it
    // We'll create a wrapper that captures nodeId and allows passing initialData
    const customField = (onRunBase?: () => void) => {
      if (!onRunBase) {
        return startCustomFieldRenderer.render(node);
      }

      // Create a wrapper onRun that can accept initialData
      // Store initialData temporarily in window and trigger the base onRun
      // handleNodeRun will check for this data
      const onRunWithContext = (initialData?: string) => {
        // Store initialData temporarily and trigger the run
        (window as any).__pendingRunData = { nodeId, initialData };
        onRunBase();
        // Clear after a short delay to allow handleNodeRun to read it
        setTimeout(() => {
          delete (window as any).__pendingRunData;
        }, 100);
      };

      return startCustomFieldRenderer.render(node, { onRun: onRunWithContext });
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
  render: (node: NodeInfo, context?: { onRun?: (initialData?: string) => void }): React.ReactNode => {
    const config = node.configuration as StartConfiguration;
    const templates = config?.templates || [];

    if (templates.length === 0) {
      return null;
    }

    const handleRun = (template: StartTemplate) => {
      if (context?.onRun) {
        const payloadString = JSON.stringify(template.payload, null, 2);
        context.onRun(payloadString);
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
            <Button
              size="sm"
              data-testid="start-template-run"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                handleRun(template);
              }}
              disabled={!context?.onRun}
              className="flex-shrink-0 h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
            >
              Run
            </Button>
          </div>
        ))}
      </div>
    );
  },
};
