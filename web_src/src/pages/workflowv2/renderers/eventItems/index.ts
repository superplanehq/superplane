import { EventItemRenderer } from "./types";
import { defaultEventItemRenderer } from "./default";
import { approvalEventItemRenderer } from "./approval";

/**
 * Registry mapping component types to their event item renderers.
 * Any component type not in this registry will use the defaultEventItemRenderer.
 */
const eventItemRenderers: Record<string, EventItemRenderer> = {
  approval: approvalEventItemRenderer,
};

/**
 * Get the appropriate event item renderer for a component type.
 * Falls back to the default renderer if no specific renderer is registered.
 */
export function getEventItemRenderer(componentType?: string): EventItemRenderer {
  return (componentType && eventItemRenderers[componentType]) || defaultEventItemRenderer;
}

// Re-export types and utilities for convenience
export * from "./types";
export { getDefaultEventItemStyle } from "./default";