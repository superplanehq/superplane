import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type {
  TriggerRenderer,
  CustomFieldRenderer,
  CustomFieldRendererContext,
  NodeInfo,
  TriggerRendererContext,
  TriggerEventContext,
} from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { renderTimeAgo } from "@/components/TimeAgo";
import React from "react";
import { Button } from "@/components/ui/button";
import { Play } from "lucide-react";
import { StartRunModal } from "./runModal";
import { payloadForTemplateRun, startRunModalTitle, type StartConfiguration } from "./templatePayload";

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
    const { node, definition, lastEvent, canvasMode, actions } = context;
    const customField = startCustomFieldRenderer.render(node, {
      canvasMode: canvasMode ?? "live",
      actions,
    });

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSlug: definition.icon || "play",
      iconColor: getColorClass("black"),
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

    const mode = context?.canvasMode ?? "live";
    const actions = context?.actions;
    const showTemplateRun = mode === "live" && !!actions;

    return (
      <div className="px-2 py-1.5 border-b border-slate-950/20 text-gray-500 flex flex-col gap-1">
        {templates.map((template, index) => (
          <div key={index} className="flex items-center justify-between min-w-0">
            <div className="flex items-center min-w-0 flex-1">
              <div className="w-4 h-4 mr-2 flex-shrink-0">
                <Play size={16} className="text-gray-500" />
              </div>
              <span className="text-[13px] font-medium font-inter text-gray-500 truncate">{template.name}</span>
            </div>
            {showTemplateRun && actions && (
              <Button
                size="xs"
                data-testid="start-template-run"
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  if ((template.parameters?.length ?? 0) > 0) {
                    actions.openModal({
                      title: startRunModalTitle(node.name, template.name),
                      content: ({ close }) => (
                        <StartRunModal
                          parameters={template.parameters}
                          initialPayload={payloadForTemplateRun(template)}
                          onClose={close}
                          onRun={async (payload) =>
                            actions.invokeNodeTriggerHook("run", {
                              template: template.name,
                              ...payload,
                            })
                          }
                        />
                      ),
                    });
                    return;
                  }
                  void actions.invokeNodeTriggerHook("run", {
                    template: template.name,
                  });
                }}
                className="flex-shrink-0 bg-black text-white hover:bg-black/80"
              >
                Run
              </Button>
            )}
          </div>
        ))}
      </div>
    );
  },
};
