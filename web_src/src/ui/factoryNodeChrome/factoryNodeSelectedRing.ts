import { cn } from "@/lib/utils";

/** Blue selection ring shared by factory run view and Configure edit. */
export const FACTORY_NODE_SELECTED_RING_CLASSNAME =
  "ring-4 ring-[color:var(--status-running-dot)] ring-offset-2 ring-offset-[color:var(--status-running-bg)]";

const FACTORY_NODE_CARD_FRAME_CLASSNAME = "group relative overflow-visible rounded-2xl";

/** Root frame classes for a factory node card, including the selection ring. */
export function factoryNodeCardFrameClassName(selected: boolean): string {
  return cn(FACTORY_NODE_CARD_FRAME_CLASSNAME, selected && "z-10", selected && FACTORY_NODE_SELECTED_RING_CLASSNAME);
}

/** `data-selected` value when the factory node card is selected. */
export function factoryNodeCardSelectedAttr(selected: boolean): "true" | undefined {
  if (!selected) {
    return undefined;
  }
  return "true";
}
