/** Storybook WorkOrderCanvas look — applied only on factory (vertical) canvases. */

export type FactoryCanvasBackground = {
  gap: number;
  size: number;
  color: string;
  bgColor: string;
};

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

/** Small square ports matching Storybook factory nodes. */
export const FACTORY_HANDLE_STYLE = {
  width: 10,
  height: 10,
  borderRadius: 3,
  border: "1px solid var(--border, #e5e7eb)",
  background: "var(--card, #ffffff)",
  boxShadow: "none",
} as const;
