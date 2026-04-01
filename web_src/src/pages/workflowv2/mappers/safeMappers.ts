import React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { TriggerProps } from "@/ui/trigger";
import type {
  ComponentAdditionalDataBuilder,
  ComponentBaseContext,
  ComponentBaseMapper,
  CustomFieldRenderer,
  TriggerRenderer,
  TriggerRendererContext,
} from "./types";

const FALLBACK_NODE_MESSAGE = "Unavailable";

type UnknownRecord = Record<string, unknown>;

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === "object" && value !== null;
}

function getFallbackComponentTitle(context: ComponentBaseContext): string {
  return context.node?.name || context.componentDefinition?.label || context.componentDefinition?.name || "Component";
}

function getFallbackTriggerTitle(context: TriggerRendererContext): string {
  return context.node?.name || context.definition?.label || context.definition?.name || "Trigger";
}

function sanitizeString(value: unknown, fallback: string = ""): string {
  return typeof value === "string" ? value : fallback;
}

function sanitizeBoolean(value: unknown, fallback: boolean = false): boolean {
  return typeof value === "boolean" ? value : fallback;
}

function sanitizeArray<T>(value: unknown): T[] | undefined {
  return Array.isArray(value) ? (value as T[]) : undefined;
}

function sanitizeReactNodeValue(node: React.ReactNode): React.ReactNode {
  if (node === null || node === undefined || typeof node === "string" || typeof node === "number") {
    return node;
  }

  if (typeof node === "boolean") {
    return null;
  }

  if (Array.isArray(node)) {
    return node.map((item, index) => React.createElement(React.Fragment, { key: index }, sanitizeReactNodeValue(item)));
  }

  if (React.isValidElement(node)) {
    return node;
  }

  return null;
}

function sanitizeCustomField(
  customField: ComponentBaseProps["customField"],
  mapperName: string,
): ComponentBaseProps["customField"] {
  if (typeof customField === "function") {
    return (onRun, nodeId) => {
      try {
        return sanitizeReactNodeValue(customField(onRun, nodeId));
      } catch (error) {
        console.error(`[SafeMapper] Component mapper "${mapperName}" threw in customField():`, error);
        return null;
      }
    };
  }

  return sanitizeReactNodeValue(customField);
}

// eslint-disable-next-line complexity
export function normalizeComponentBaseProps(
  props: ComponentBaseProps | unknown,
  context: ComponentBaseContext,
): ComponentBaseProps {
  const fallbackTitle = getFallbackComponentTitle(context);
  const fallbackIconSlug = context.componentDefinition?.icon || "circle-off";
  const fallbackEmptyStateProps: NonNullable<ComponentBaseProps["emptyStateProps"]> = {
    icon: undefined,
    title: FALLBACK_NODE_MESSAGE,
    description: undefined,
  };
  const record = isRecord(props) ? props : {};
  const normalized = {
    ...record,
    iconSrc: typeof record.iconSrc === "string" ? record.iconSrc : undefined,
    iconSlug: sanitizeString(record.iconSlug, fallbackIconSlug),
    iconColor: typeof record.iconColor === "string" ? record.iconColor : undefined,
    title: sanitizeString(record.title, fallbackTitle),
    showHeader: typeof record.showHeader === "boolean" ? record.showHeader : undefined,
    paused: typeof record.paused === "boolean" ? record.paused : undefined,
    specs: sanitizeArray(record.specs),
    hideCount: typeof record.hideCount === "boolean" ? record.hideCount : undefined,
    hideMetadataList: typeof record.hideMetadataList === "boolean" ? record.hideMetadataList : undefined,
    collapsed: sanitizeBoolean(record.collapsed, context.node?.isCollapsed ?? false),
    collapsedBackground: typeof record.collapsedBackground === "string" ? record.collapsedBackground : undefined,
    eventSections: sanitizeArray(record.eventSections),
    selected: typeof record.selected === "boolean" ? record.selected : undefined,
    metadata: sanitizeArray(record.metadata),
    customField: sanitizeCustomField(record.customField as ComponentBaseProps["customField"], fallbackTitle),
    customFieldPosition: record.customFieldPosition === "before" ? "before" : "after",
    eventStateMap: isRecord(record.eventStateMap)
      ? (record.eventStateMap as ComponentBaseProps["eventStateMap"])
      : undefined,
    includeEmptyState: sanitizeBoolean(record.includeEmptyState, false),
    emptyStateProps: isRecord(record.emptyStateProps)
      ? ({
          icon:
            typeof record.emptyStateProps.icon === "function"
              ? (record.emptyStateProps.icon as NonNullable<ComponentBaseProps["emptyStateProps"]>["icon"])
              : undefined,
          title: sanitizeString(record.emptyStateProps.title),
          description: sanitizeString(record.emptyStateProps.description),
        } as NonNullable<ComponentBaseProps["emptyStateProps"]>)
      : undefined,
    error: sanitizeString(record.error),
    warning: sanitizeString(record.warning),
  } satisfies ComponentBaseProps;

  const isFallback = !isRecord(props) || typeof record.title !== "string";
  if (!normalized.title) {
    normalized.title = fallbackTitle;
  }
  if (!normalized.iconSlug) {
    normalized.iconSlug = fallbackIconSlug;
  }
  if (isFallback) {
    normalized.includeEmptyState = true;
    normalized.emptyStateProps = normalized.emptyStateProps || fallbackEmptyStateProps;
  }

  return normalized;
}

// eslint-disable-next-line complexity
export function normalizeTriggerProps(props: TriggerProps | unknown, context: TriggerRendererContext): TriggerProps {
  const fallbackProps = {
    title: getFallbackTriggerTitle(context),
    iconSlug: context.definition?.icon || "bolt",
    metadata: [],
  } satisfies TriggerProps;
  const record = isRecord(props) ? props : {};
  const normalizedComponentProps = normalizeComponentBaseProps(
    {
      ...record,
      title: sanitizeString(record.title, fallbackProps.title),
      iconSlug: sanitizeString(record.iconSlug, fallbackProps.iconSlug),
      metadata: sanitizeArray(record.metadata),
    },
    {
      nodes: [],
      node: context.node,
      componentDefinition: {
        name: context.definition?.name || "",
        label: context.definition?.label || fallbackProps.title,
        description: context.definition?.description || "",
        icon: context.definition?.icon || fallbackProps.iconSlug,
        color: context.definition?.color || "",
      },
      lastExecutions: [],
    },
  );

  const lastEventData = isRecord(record.lastEventData)
    ? {
        title: sanitizeString(record.lastEventData.title, fallbackProps.title),
        subtitle:
          typeof record.lastEventData.subtitle === "string" || React.isValidElement(record.lastEventData.subtitle)
            ? (record.lastEventData.subtitle as NonNullable<TriggerProps["lastEventData"]>["subtitle"])
            : undefined,
        receivedAt: record.lastEventData.receivedAt instanceof Date ? record.lastEventData.receivedAt : new Date(),
        state: sanitizeString(record.lastEventData.state, "triggered"),
        eventId: sanitizeString(record.lastEventData.eventId),
      }
    : undefined;

  return {
    ...normalizedComponentProps,
    title: sanitizeString(record.title, fallbackProps.title),
    iconSlug: sanitizeString(record.iconSlug, fallbackProps.iconSlug),
    metadata: sanitizeArray(record.metadata) || [],
    lastEventData,
  };
}

/**
 * Wraps a ComponentBaseMapper so that any exception thrown by its methods
 * is caught, logged, and replaced with a safe fallback value.
 *
 * This is the frontend equivalent of the PanicableComponent pattern used
 * in the backend (pkg/registry/component.go) to prevent a single mapper
 * failure from breaking the entire canvas.
 */
export function createSafeComponentMapper(mapper: ComponentBaseMapper, mapperName: string): ComponentBaseMapper {
  return {
    props(context) {
      try {
        return normalizeComponentBaseProps(mapper.props(context), context);
      } catch (error) {
        console.error(`[SafeMapper] Component mapper "${mapperName}" threw in props():`, error);
        const fallbackProps: ComponentBaseProps = {
          iconSlug: context.componentDefinition?.icon ?? "circle-off",
          collapsed: context.node?.isCollapsed ?? false,
          title:
            context.node?.name ||
            context.componentDefinition?.label ||
            context.componentDefinition?.name ||
            "Component",
          includeEmptyState: true,
          emptyStateProps: {
            title: FALLBACK_NODE_MESSAGE,
            description: undefined,
          },
        };
        return fallbackProps;
      }
    },

    subtitle(context) {
      try {
        return mapper.subtitle(context);
      } catch (error) {
        console.error(`[SafeMapper] Component mapper "${mapperName}" threw in subtitle():`, error);
        return "";
      }
    },

    getExecutionDetails(context) {
      try {
        return mapper.getExecutionDetails(context);
      } catch (error) {
        console.error(`[SafeMapper] Component mapper "${mapperName}" threw in getExecutionDetails():`, error);
        return {};
      }
    },
  };
}

/**
 * Wraps a TriggerRenderer so that any exception thrown by its methods
 * is caught, logged, and replaced with a safe fallback value.
 *
 * This is the frontend equivalent of the PanicableTrigger pattern used
 * in the backend (pkg/registry/trigger.go) to prevent a single renderer
 * failure from breaking the entire canvas.
 */
export function createSafeTriggerRenderer(renderer: TriggerRenderer, rendererName: string): TriggerRenderer {
  return {
    getTriggerProps(context) {
      try {
        return normalizeTriggerProps(renderer.getTriggerProps(context), context);
      } catch (error) {
        console.error(`[SafeMapper] Trigger renderer "${rendererName}" threw in getTriggerProps():`, error);
        const fallbackProps: TriggerProps = {
          title: context.node?.name || context.definition?.label || "Trigger",
          iconSlug: context.definition?.icon || "bolt",
          metadata: [],
          includeEmptyState: true,
          emptyStateProps: {
            title: FALLBACK_NODE_MESSAGE,
            description: undefined,
          },
        };
        return fallbackProps;
      }
    },

    getRootEventValues(context) {
      try {
        return renderer.getRootEventValues(context);
      } catch (error) {
        console.error(`[SafeMapper] Trigger renderer "${rendererName}" threw in getRootEventValues():`, error);
        return {};
      }
    },

    getTitleAndSubtitle(context) {
      try {
        return renderer.getTitleAndSubtitle(context);
      } catch (error) {
        console.error(`[SafeMapper] Trigger renderer "${rendererName}" threw in getTitleAndSubtitle():`, error);
        return { title: "Event", subtitle: "" };
      }
    },

    getEventState: renderer.getEventState
      ? (context) => {
          try {
            return renderer.getEventState!(context);
          } catch (error) {
            console.error(`[SafeMapper] Trigger renderer "${rendererName}" threw in getEventState():`, error);
            return "triggered";
          }
        }
      : undefined,
  };
}

export function createSafeAdditionalDataBuilder(
  builder: ComponentAdditionalDataBuilder,
  builderName: string,
): ComponentAdditionalDataBuilder {
  return {
    buildAdditionalData(context) {
      try {
        return builder.buildAdditionalData(context);
      } catch (error) {
        console.error(`[SafeMapper] Additional data builder "${builderName}" threw in buildAdditionalData():`, error);
        return undefined;
      }
    },
  };
}

export function createSafeCustomFieldRenderer(
  renderer: CustomFieldRenderer,
  rendererName: string,
): CustomFieldRenderer {
  return {
    render(node, context) {
      try {
        return sanitizeReactNodeValue(renderer.render(node, context));
      } catch (error) {
        console.error(`[SafeMapper] Custom field renderer "${rendererName}" threw in render():`, error);
        return null;
      }
    },
  };
}
