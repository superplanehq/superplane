import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";

interface OnInvokeParameterDefinition {
  name?: string;
  label?: string;
}

interface OnInvokeConfiguration {
  parameters?: OnInvokeParameterDefinition[];
}

interface OnInvokeEventData {
  app?: {
    id?: string;
    name?: string;
  };
  payload?: Record<string, unknown>;
}

export const onInvokeTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnInvokeEventData | undefined;

    return {
      title: invokeTitle(eventData),
      subtitle: "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnInvokeEventData | undefined;
    const values: Record<string, string> = {};

    if (eventData?.app?.name) {
      values.App = eventData.app.name;
    } else if (eventData?.app?.id) {
      values.App = eventData.app.id;
    }

    if (eventData?.payload && typeof eventData.payload === "object" && !Array.isArray(eventData.payload)) {
      for (const [key, value] of Object.entries(eventData.payload)) {
        const formatted = formatEventValue(value);
        if (formatted.length > 0) {
          values[key] = formatted;
        }
      }
    }

    if (context.event?.createdAt) {
      values["Received at"] = new Date(context.event.createdAt).toLocaleString();
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnInvokeConfiguration | undefined;
    const parameterCount = configuration?.parameters?.length ?? 0;

    const props: TriggerProps = {
      title: node.name || definition.label || "On Invoke",
      iconSlug: definition.icon || "play",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata:
        parameterCount > 0
          ? [{ icon: "list", label: `${parameterCount} parameter${parameterCount === 1 ? "" : "s"}` }]
          : [],
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnInvokeEventData | undefined;

      props.lastEventData = {
        title: invokeTitle(eventData),
        subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

export function invokeTitle(eventData: OnInvokeEventData | undefined): string {
  const appName = eventData?.app?.name?.trim();
  if (appName) {
    return `Invoked from ${appName}`;
  }

  return "App invoked";
}

function formatEventValue(value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }

  if (typeof value === "string") {
    return value;
  }

  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  return JSON.stringify(value);
}
