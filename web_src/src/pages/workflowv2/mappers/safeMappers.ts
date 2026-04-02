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

type UnknownRecord = Record<string, unknown>;

export const CANVAS_NODE_FALLBACK_MESSAGE = "Can't display";

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === "boolean" ? value : undefined;
}

function getComponentTitle(context: ComponentBaseContext): string {
  return context.node?.name || context.componentDefinition?.label || context.componentDefinition?.name || "Component";
}

function getTriggerTitle(context: TriggerRendererContext): string {
  return context.node?.name || context.definition?.label || context.definition?.name || "Trigger";
}

function sanitizeString(value: unknown, fallback: string = ""): string {
  return typeof value === "string" ? value : fallback;
}

function sanitizeNonEmptyString(value: unknown, fallback: string = ""): string {
  return typeof value === "string" && value.trim() ? value : fallback;
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

function normalizeEmptyStateProps(
  emptyStateProps: unknown,
): NonNullable<ComponentBaseProps["emptyStateProps"]> | undefined {
  if (!isRecord(emptyStateProps)) {
    return undefined;
  }

  return {
    icon:
      typeof emptyStateProps.icon === "function"
        ? (emptyStateProps.icon as NonNullable<ComponentBaseProps["emptyStateProps"]>["icon"])
        : undefined,
    title: sanitizeString(emptyStateProps.title),
    description: sanitizeString(emptyStateProps.description),
  };
}

function buildNormalizedComponentBaseProps(
  record: UnknownRecord,
  context: ComponentBaseContext,
  fallbackTitle: string,
  fallbackIconSlug: string,
): ComponentBaseProps {
  return {
    ...record,
    iconSrc: asString(record.iconSrc),
    iconSlug: sanitizeString(record.iconSlug, fallbackIconSlug),
    iconColor: asString(record.iconColor),
    title: sanitizeString(record.title, fallbackTitle),
    showHeader: asBoolean(record.showHeader),
    paused: asBoolean(record.paused),
    specs: sanitizeArray(record.specs),
    hideCount: asBoolean(record.hideCount),
    hideMetadataList: asBoolean(record.hideMetadataList),
    collapsed: sanitizeBoolean(record.collapsed, context.node?.isCollapsed ?? false),
    collapsedBackground: asString(record.collapsedBackground),
    eventSections: sanitizeArray(record.eventSections),
    selected: asBoolean(record.selected),
    metadata: sanitizeArray(record.metadata),
    customField: sanitizeCustomField(record.customField as ComponentBaseProps["customField"], fallbackTitle),
    customFieldPosition: record.customFieldPosition === "before" ? "before" : "after",
    eventStateMap: isRecord(record.eventStateMap)
      ? (record.eventStateMap as ComponentBaseProps["eventStateMap"])
      : undefined,
    includeEmptyState: sanitizeBoolean(record.includeEmptyState, false),
    emptyStateProps: normalizeEmptyStateProps(record.emptyStateProps),
    error: sanitizeString(record.error),
    warning: sanitizeString(record.warning),
  };
}

function applyComponentBaseFallbacks(
  normalized: ComponentBaseProps,
  props: ComponentBaseProps | unknown,
  record: UnknownRecord,
  fallbackTitle: string,
  fallbackIconSlug: string,
): ComponentBaseProps {
  const fallbackEmptyStateProps: NonNullable<ComponentBaseProps["emptyStateProps"]> = {
    icon: undefined,
    title: CANVAS_NODE_FALLBACK_MESSAGE,
    description: undefined,
  };
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

export function normalizeComponentBaseProps(
  props: ComponentBaseProps | unknown,
  context: ComponentBaseContext,
): ComponentBaseProps {
  const fallbackTitle = getComponentTitle(context);
  const fallbackIconSlug = context.componentDefinition?.icon || "circle-off";
  const record = isRecord(props) ? props : {};
  const normalized = buildNormalizedComponentBaseProps(record, context, fallbackTitle, fallbackIconSlug);

  return applyComponentBaseFallbacks(normalized, props, record, fallbackTitle, fallbackIconSlug);
}

function buildFallbackTriggerProps(context: TriggerRendererContext): TriggerProps {
  return {
    title: getTriggerTitle(context),
    iconSlug: context.definition?.icon || "bolt",
    metadata: [],
  };
}

function buildLastEventData(lastEventData: unknown, fallbackTitle: string): TriggerProps["lastEventData"] {
  if (!isRecord(lastEventData)) {
    return undefined;
  }

  const subtitle = lastEventData.subtitle;

  return {
    title: sanitizeNonEmptyString(lastEventData.title, fallbackTitle),
    subtitle:
      typeof subtitle === "string" || React.isValidElement(subtitle)
        ? (subtitle as NonNullable<TriggerProps["lastEventData"]>["subtitle"])
        : undefined,
    receivedAt: lastEventData.receivedAt instanceof Date ? lastEventData.receivedAt : new Date(),
    state: sanitizeString(lastEventData.state, "triggered"),
    eventId: sanitizeString(lastEventData.eventId),
  };
}

function normalizeTriggerCustomField(customField: unknown): ComponentBaseProps["customField"] {
  if (typeof customField === "function") {
    return (onRun, nodeId) => {
      try {
        const fn = customField as (onRun?: () => void, nodeId?: string) => React.ReactNode;
        return sanitizeReactNodeValue(fn(onRun, nodeId));
      } catch (error) {
        console.error("[SafeMapper] Trigger customField() threw:", error);
        return null;
      }
    };
  }

  return sanitizeReactNodeValue(customField as React.ReactNode);
}

function buildNormalizedTriggerProps(
  record: UnknownRecord,
  context: TriggerRendererContext,
  fallbackProps: TriggerProps,
): TriggerProps {
  const metadata = sanitizeArray<TriggerProps["metadata"][number]>(record.metadata) || [];
  const normalizedTitle = sanitizeNonEmptyString(record.title, fallbackProps.title);
  const normalizedIconSlug = sanitizeNonEmptyString(record.iconSlug, fallbackProps.iconSlug);

  return {
    ...record,
    iconSrc: asString(record.iconSrc),
    iconSlug: normalizedIconSlug,
    iconColor: asString(record.iconColor),
    title: normalizedTitle,
    showHeader: asBoolean(record.showHeader),
    paused: asBoolean(record.paused),
    specs: sanitizeArray(record.specs),
    hideCount: asBoolean(record.hideCount),
    hideMetadataList: asBoolean(record.hideMetadataList),
    collapsed: sanitizeBoolean(record.collapsed, context.node?.isCollapsed ?? false),
    collapsedBackground: asString(record.collapsedBackground),
    selected: asBoolean(record.selected),
    metadata,
    customField: normalizeTriggerCustomField(record.customField),
    customFieldPosition: record.customFieldPosition === "before" ? "before" : "after",
    eventStateMap: isRecord(record.eventStateMap)
      ? (record.eventStateMap as ComponentBaseProps["eventStateMap"])
      : undefined,
    includeEmptyState: sanitizeBoolean(record.includeEmptyState, false),
    emptyStateProps: normalizeEmptyStateProps(record.emptyStateProps),
    error: sanitizeString(record.error),
    warning: sanitizeString(record.warning),
    lastEventData: buildLastEventData(record.lastEventData, normalizedTitle),
  };
}

function applyTriggerFallbacks(
  normalized: TriggerProps,
  props: TriggerProps | unknown,
  record: UnknownRecord,
): TriggerProps {
  if (isRecord(props) && typeof record.title === "string") {
    return normalized;
  }

  return {
    ...normalized,
    includeEmptyState: true,
    emptyStateProps: normalized.emptyStateProps || {
      icon: undefined,
      title: CANVAS_NODE_FALLBACK_MESSAGE,
      description: undefined,
    },
  };
}

export function normalizeTriggerProps(props: TriggerProps | unknown, context: TriggerRendererContext): TriggerProps {
  const fallbackProps = buildFallbackTriggerProps(context);
  const record = isRecord(props) ? props : {};
  const normalizedTriggerProps = buildNormalizedTriggerProps(record, context, fallbackProps);

  return applyTriggerFallbacks(normalizedTriggerProps, props, record);
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
            title: CANVAS_NODE_FALLBACK_MESSAGE,
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
            title: CANVAS_NODE_FALLBACK_MESSAGE,
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
