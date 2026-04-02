import { isRecord } from "@/lib/records";

const CANVAS_NODE_FALLBACK_MESSAGE = "Can't display";
import type { AnnotationComponentProps } from "../../annotationComponent";
import type { ComponentBaseProps } from "../../componentBase";
import type { CompositeProps } from "../../composite";
import type { TriggerProps } from "../../trigger";
import type { BlockData } from "./types";

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
  const message = data.renderFallback?.message || CANVAS_NODE_FALLBACK_MESSAGE;
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

  const component = data.component!;
  return {
    ...component,
    title:
      typeof component.title === "string" && component.title.trim()
        ? component.title
        : getBlockLabel(data, "Component"),
    error: typeof component.error === "string" ? component.error : "",
    warning: typeof component.warning === "string" ? component.warning : "",
    metadata: Array.isArray(component.metadata) ? component.metadata : undefined,
    specs: Array.isArray(component.specs) ? component.specs : undefined,
    eventSections: Array.isArray(component.eventSections) ? component.eventSections : undefined,
    iconSlug: typeof component.iconSlug === "string" ? component.iconSlug : "box",
    collapsed: typeof component.collapsed === "boolean" ? component.collapsed : false,
    includeEmptyState: typeof component.includeEmptyState === "boolean" ? component.includeEmptyState : false,
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

  const trigger = data.trigger!;
  return {
    ...trigger,
    title:
      typeof trigger.title === "string" && trigger.title.trim()
        ? trigger.title
        : getBlockLabel(data, "Trigger"),
    iconSlug: typeof trigger.iconSlug === "string" ? trigger.iconSlug : "bolt",
    metadata: Array.isArray(trigger.metadata) ? trigger.metadata : [],
    error: typeof trigger.error === "string" ? trigger.error : "",
    warning: typeof trigger.warning === "string" ? trigger.warning : "",
  };
}

export function getSafeCompositeProps(data: BlockData): CompositeProps {
  if (!isRecord(data.composite)) {
    return {
      ...buildFallbackComponentProps(data, "Composite"),
      title: getBlockLabel(data, "Composite"),
    };
  }

  const composite = data.composite!;
  return {
    ...composite,
    title:
      typeof composite.title === "string" && composite.title.trim()
        ? composite.title
        : getBlockLabel(data, "Composite"),
    metadata: Array.isArray(composite.metadata) ? composite.metadata : undefined,
    parameters: Array.isArray(composite.parameters) ? composite.parameters : [],
    iconSlug: typeof composite.iconSlug === "string" ? composite.iconSlug : "component",
    collapsed: typeof composite.collapsed === "boolean" ? composite.collapsed : false,
    error: typeof composite.error === "string" ? composite.error : "",
    warning: typeof composite.warning === "string" ? composite.warning : "",
  };
}

export function getSafeAnnotationProps(data: BlockData): AnnotationComponentProps | null {
  if (!isRecord(data.annotation)) {
    return null;
  }

  const annotation = data.annotation!;
  return {
    ...annotation,
    title:
      typeof annotation.title === "string" && annotation.title.trim()
        ? annotation.title
        : getBlockLabel(data, "Annotation"),
    annotationText: typeof annotation.annotationText === "string" ? annotation.annotationText : "",
    annotationColor: typeof annotation.annotationColor === "string" ? annotation.annotationColor : "yellow",
    width: typeof annotation.width === "number" ? annotation.width : 320,
    height: typeof annotation.height === "number" ? annotation.height : 200,
  };
}
