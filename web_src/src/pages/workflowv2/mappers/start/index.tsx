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

interface StartTemplate {
  name: string;
  payload: Record<string, unknown>;
}

interface StartConfiguration {
  templates?: StartTemplate[];
}

function payloadForTemplateRun(template: StartTemplate): Record<string, unknown> {
  const p = template.payload;
  if (p && typeof p === "object" && !Array.isArray(p)) {
    return p as Record<string, unknown>;
  }
  return {};
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
    const { node, definition, lastEvent, canvasMode, actions } = context;
    const customField = startCustomFieldRenderer.render(node, {
      canvasMode: canvasMode ?? "live",
      actions,
    });

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
 * Custom field renderer for the start trigger that displays templates with Run and Edit buttons
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

    const onRunTemplateClick = async (e: React.MouseEvent<HTMLButtonElement>, template: StartTemplate) => {
      e.preventDefault();
      e.stopPropagation();
      if(!actions) return;
      await actions.invokeNodeTriggerHook("run", {
        template: template.name,
        payload: payloadForTemplateRun(template),
      });
    }

    const onEditTemplateClick = (e: React.MouseEvent<HTMLButtonElement>, template: StartTemplate) => {
      e.preventDefault();
      e.stopPropagation();
      if(!actions) return;
      actions.openModal({
        title: "Run trigger",
        description: (
          <>
            Run template <strong>{template.name}</strong> on node{" "}
            <strong>{node.name || "Unnamed trigger"}</strong>. Edit the payload below to override the
            template default.
          </>
        ),
        content: ({ close }) => (
          <StartRunModal
            initialPayload={payloadForTemplateRun(template)}
            onClose={close}
            onRun={async (payload) => {
              await actions.invokeNodeTriggerHook("run", {
                template: template.name,
                payload,
              });
            }}
          />
        ),
      });
      }



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
            {showTemplateRun && actions && (
              <div className="flex items-center gap-1 flex-shrink-0">
                <Button
                  size="sm"
                  variant="outline"
                  data-testid="start-template-edit"
                  onClick={(e) => onEditTemplateClick(e, template)}
                  className="h-7 py-1 px-2"
                >
                  Edit
                </Button>
                <Button
                  size="sm"
                  data-testid="start-template-run"
                  onClick={(e) => onRunTemplateClick(e, template)}
                  className="h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
                >
                  Run
                </Button>
              </div>
            )}
          </div>
        ))}
      </div>
    );
  },
};
