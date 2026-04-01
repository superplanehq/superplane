import type { AnnotationComponentProps } from "../../annotationComponent";
import type { ComponentBaseProps } from "../../componentBase";
import type { CompositeProps } from "../../composite";
import type { TriggerProps } from "../../trigger";
import type { BlockData, UnknownRecord } from "./types";

export const FALLBACK_NODE_MESSAGE = "Can't display";

export function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === "object" && value !== null;
}

export function getBlockLabel(data: BlockData, fallback: string): string {
  return typeof data.label === "string" && data.label.trim() ? data.label : fallback;
}

export function getOutputChannels(data: BlockData): string[] {
  const channels = Array.isArray(data.outputChannels)
    ? data.outputChannels.filter((channel) => typeof channel === "string")
    : [];
  return channels.length > 0 ? channels : ["default"];
}

export function buildFallbackComponentProps(data: BlockData, fallbackTitle: string): ComponentBaseProps {
  const message = data.renderFallback?.message || FALLBACK_NODE_MESSAGE;
  return {
    iconSlug: "triangle-alert",
    collapsed: false,
    title: getBlockLabel(data, fallbackTitle),
    includeEmptyState: true,
    emptyStateProps: {
      title: message,
      description: undefined,
    },
  };
}

export function getSafeComponentProps(data: BlockData): ComponentBaseProps {
  if (!isRecord(data.component)) {
    return buildFallbackComponentProps(data, "Component");
  }

  return {
    ...(data.component as ComponentBaseProps),
    title:
      typeof data.component.title === "string" && data.component.title.trim()
        ? data.component.title
        : getBlockLabel(data, "Component"),
    error: typeof data.component.error === "string" ? data.component.error : "",
    warning: typeof data.component.warning === "string" ? data.component.warning : "",
    metadata: Array.isArray(data.component.metadata) ? data.component.metadata : undefined,
    specs: Array.isArray(data.component.specs) ? data.component.specs : undefined,
    eventSections: Array.isArray(data.component.eventSections) ? data.component.eventSections : undefined,
    iconSlug: typeof data.component.iconSlug === "string" ? data.component.iconSlug : "box",
    collapsed: typeof data.component.collapsed === "boolean" ? data.component.collapsed : false,
    includeEmptyState: typeof data.component.includeEmptyState === "boolean" ? data.component.includeEmptyState : false,
  };
}

export function getSafeTriggerProps(data: BlockData): TriggerProps {
  if (!isRecord(data.trigger)) {
    return {
      ...buildFallbackComponentProps(data, "Trigger"),
      title: getBlockLabel(data, "Trigger"),
      iconSlug: "bolt",
      metadata: [],
    };
  }

  return {
    ...(data.trigger as TriggerProps),
    title:
      typeof data.trigger.title === "string" && data.trigger.title.trim()
        ? data.trigger.title
        : getBlockLabel(data, "Trigger"),
    iconSlug: typeof data.trigger.iconSlug === "string" ? data.trigger.iconSlug : "bolt",
    metadata: Array.isArray(data.trigger.metadata) ? data.trigger.metadata : [],
    error: typeof data.trigger.error === "string" ? data.trigger.error : "",
    warning: typeof data.trigger.warning === "string" ? data.trigger.warning : "",
  };
}

export function getSafeCompositeProps(data: BlockData): CompositeProps {
  if (!isRecord(data.composite)) {
    return {
      ...buildFallbackComponentProps(data, "Composite"),
      title: getBlockLabel(data, "Composite"),
    };
  }

  return {
    ...(data.composite as CompositeProps),
    title:
      typeof data.composite.title === "string" && data.composite.title.trim()
        ? data.composite.title
        : getBlockLabel(data, "Composite"),
    metadata: Array.isArray(data.composite.metadata) ? data.composite.metadata : undefined,
    parameters: Array.isArray(data.composite.parameters) ? data.composite.parameters : [],
    iconSlug: typeof data.composite.iconSlug === "string" ? data.composite.iconSlug : "component",
    collapsed: typeof data.composite.collapsed === "boolean" ? data.composite.collapsed : false,
    error: typeof data.composite.error === "string" ? data.composite.error : "",
    warning: typeof data.composite.warning === "string" ? data.composite.warning : "",
  };
}

export function getSafeAnnotationProps(data: BlockData): AnnotationComponentProps | null {
  if (!isRecord(data.annotation)) {
    return null;
  }

  return {
    ...(data.annotation as AnnotationComponentProps),
    title:
      typeof data.annotation.title === "string" && data.annotation.title.trim()
        ? data.annotation.title
        : getBlockLabel(data, "Annotation"),
    annotationText: typeof data.annotation.annotationText === "string" ? data.annotation.annotationText : "",
    annotationColor: typeof data.annotation.annotationColor === "string" ? data.annotation.annotationColor : "yellow",
    width: typeof data.annotation.width === "number" ? data.annotation.width : 320,
    height: typeof data.annotation.height === "number" ? data.annotation.height : 200,
  };
}
