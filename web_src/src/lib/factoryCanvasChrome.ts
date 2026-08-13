/** Storybook WorkOrderCanvas look — applied only on factory (vertical) canvases. */

export type FactoryCanvasBackground = {
  gap: number;
  size: number;
  color: string;
  bgColor: string;
};

export type FactoryEdgeTone = "default" | "running" | "failed";

export type FactoryEdgePalette = {
  default: { stroke: string; strokeWidth: number };
  running: { stroke: string; strokeWidth: number };
  failed: { stroke: string; strokeWidth: number };
};

export function factoryCanvasBackground(isDark: boolean): FactoryCanvasBackground {
  if (isDark) {
    return { gap: 22, size: 1, color: "#33312b", bgColor: "#14120b" };
  }
  return { gap: 22, size: 1, color: "#e5e7eb", bgColor: "#f9fafb" };
}

export function factoryEdgePalette(isDark: boolean): FactoryEdgePalette {
  if (isDark) {
    return {
      default: { stroke: "#4a4740", strokeWidth: 1.5 },
      running: { stroke: "#818cf8", strokeWidth: 1.5 },
      failed: { stroke: "#f87171", strokeWidth: 1.5 },
    };
  }
  return {
    default: { stroke: "#cbd5e1", strokeWidth: 1.5 },
    running: { stroke: "#818cf8", strokeWidth: 1.5 },
    failed: { stroke: "#fca5a5", strokeWidth: 1.5 },
  };
}

/** Map node event state → edge tone (target node drives the edge, like Storybook). */
export function resolveFactoryEdgeTone(eventState: string | undefined): FactoryEdgeTone {
  switch (eventState) {
    case "failed":
    case "error":
      return "failed";
    case "running":
    case "cancelling":
      return "running";
    // queued / cancelled / triggered / success stay default slate
    default:
      return "default";
  }
}

export function factoryEdgeToneClassName(tone: FactoryEdgeTone): string | undefined {
  if (tone === "running") return "sp-edge-factory-running";
  if (tone === "failed") return "sp-edge-factory-failed";
  return undefined;
}

/** Read primary event state from canvas block node data (component or trigger). */
export function primaryEventStateFromCanvasNodeData(data: unknown): string | undefined {
  if (!data || typeof data !== "object") return undefined;

  const record = data as Record<string, unknown>;
  for (const key of ["component", "trigger"] as const) {
    const block = record[key];
    if (!block || typeof block !== "object") continue;

    const sections = (block as { eventSections?: Array<{ eventState?: unknown }> }).eventSections;
    const state = sections?.[0]?.eventState;
    if (typeof state === "string" && state.length > 0) return state;
  }

  return undefined;
}

/** Small square ports matching Storybook factory nodes. */
export const FACTORY_HANDLE_STYLE = {
  width: 10,
  height: 10,
  borderRadius: 3,
  border: "1px solid var(--border, #e5e7eb)",
  background: "var(--card, #ffffff)",
  boxShadow: "none",
} as const;

/** Keep FactoryNodeCard, append ghost, placement gap, and ELK estimates aligned. */
export const FACTORY_NODE_CARD_WIDTH = 280;
export const FACTORY_NODE_CARD_HEIGHT = 104;
/** Vertical gap below a factory card when appending / layering. */
export const FACTORY_NODE_VERTICAL_GAP = 64;
