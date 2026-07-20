import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";

interface OnRunParameterDefinition {
  name?: string;
  label?: string;
}

interface OnRunConfiguration {
  parameters?: OnRunParameterDefinition[];
}

interface OnRunEventData {
  app?: {
    id?: string;
    name?: string;
  };
  parameters?: Record<string, unknown>;
}

export const onRunTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnRunEventData | undefined;

    return {
      title: onRunTitle(eventData),
      subtitle: "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnRunEventData | undefined;
    const values: Record<string, string> = {};

    if (eventData?.app?.name) {
      values.App = eventData.app.name;
    } else if (eventData?.app?.id) {
      values.App = eventData.app.id;
    }

    if (eventData?.parameters && typeof eventData.parameters === "object" && !Array.isArray(eventData.parameters)) {
      for (const [key, value] of Object.entries(eventData.parameters)) {
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
    const configuration = node.configuration as OnRunConfiguration | undefined;
    const parameterCount = configuration?.parameters?.length ?? 0;

    const props: TriggerProps = {
      title: node.name || definition.label || "On Run",
      iconSlug: definition.icon || "play",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata:
        parameterCount > 0
          ? [{ icon: "list", label: `${parameterCount} parameter${parameterCount === 1 ? "" : "s"}` }]
          : [],
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnRunEventData | undefined;

      props.lastEventData = {
        title: onRunTitle(eventData),
        subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

export function onRunTitle(eventData: OnRunEventData | undefined): string {
  const appName = eventData?.app?.name?.trim();
  if (appName) {
    return `Run from ${appName}`;
  }

  return "App run";
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
