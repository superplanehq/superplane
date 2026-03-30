import type { ComponentBaseProps } from "@/ui/componentBase";
import type { TriggerProps } from "@/ui/trigger";
import type { ComponentBaseMapper, TriggerRenderer } from "./types";

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
        return mapper.props(context);
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
        return renderer.getTriggerProps(context);
      } catch (error) {
        console.error(`[SafeMapper] Trigger renderer "${rendererName}" threw in getTriggerProps():`, error);
        const fallbackProps: TriggerProps = {
          title: context.node?.name || context.definition?.label || "Trigger",
          iconSlug: context.definition?.icon || "bolt",
          metadata: [],
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
