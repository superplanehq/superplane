import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type {
  TriggerRenderer,
  CustomFieldRenderer,
  CustomFieldRendererContext,
  NodeInfo,
  TriggerRendererContext,
  TriggerEventContext,
} from "./types";
import type { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { showErrorToast } from "@/lib/toast";
import { renderTimeAgo } from "@/components/TimeAgo";
import React from "react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import Editor from "@monaco-editor/react";
import { Play } from "lucide-react";

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
              <Button
                size="sm"
                data-testid="start-template-run"
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
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
                }}
                className="flex-shrink-0 h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
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

function StartRunModal({
  initialPayload,
  onRun,
  onClose,
}: {
  initialPayload: Record<string, unknown>;
  onRun: (payload: Record<string, unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const [eventData, setEventData] = React.useState<string>(() => JSON.stringify(initialPayload, null, 2));
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = async () => {
    let parsedData: Record<string, unknown>;
    try {
      const candidate = JSON.parse(eventData) as unknown;
      if (!candidate || typeof candidate !== "object" || Array.isArray(candidate)) {
        showErrorToast("Payload must be a JSON object");
        return;
      }
      parsedData = candidate as Record<string, unknown>;
    } catch {
      showErrorToast("Invalid JSON format");
      return;
    }

    setIsSubmitting(true);
    try {
      await onRun(parsedData);
      onClose();
    } catch {
      // Keep the modal open so users can retry with the same payload.
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="border border-gray-200 dark:border-gray-700 rounded-md overflow-hidden">
        <Editor
          height="300px"
          defaultLanguage="json"
          value={eventData}
          onChange={(value) => setEventData(value || "{}")}
          options={{
            minimap: { enabled: false },
            fontSize: 13,
            lineNumbers: "on",
            scrollBeyondLastLine: false,
            automaticLayout: true,
          }}
        />
      </div>
      <div className="flex items-center justify-end gap-2">
        <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
          Cancel
        </Button>
        <LoadingButton
          data-testid="emit-event-submit-button"
          loading={isSubmitting}
          loadingText="Running..."
          onClick={handleSubmit}
        >
          Run
        </LoadingButton>
      </div>
    </div>
  );
}
