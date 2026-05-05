import { createContext, type ReactNode, useContext, useState } from "react";

type SetNode = (node: ReactNode) => void;

// Two contexts so the registering side (LaunchpadView, RunViewToggle, ...)
// only consumes a stable setter and never re-renders when the registered
// node changes; the rendering side (Header) consumes the node value.
// Combining both into one context with `{ node, setNode }` would cause the
// registering effect to re-fire every time `setNode` is called, which loops
// infinitely.
const HeaderActionSlotSetterContext = createContext<SetNode | null>(null);
const HeaderActionSlotNodeContext = createContext<ReactNode>(null);

export function HeaderActionSlotProvider({ children }: { children: ReactNode }) {
  const [node, setNode] = useState<ReactNode>(null);
  return (
    <HeaderActionSlotSetterContext.Provider value={setNode}>
      <HeaderActionSlotNodeContext.Provider value={node}>{children}</HeaderActionSlotNodeContext.Provider>
    </HeaderActionSlotSetterContext.Provider>
  );
}

// Returns a stable setter for the secondary-header action slot, or null when
// no provider is mounted (allowing standalone consumers to fall back
// gracefully).
export function useHeaderActionSlotSetter() {
  return useContext(HeaderActionSlotSetterContext);
}

// Returns the current node registered in the slot. Only the Header should
// read this -- re-renders on node changes.
export function useHeaderActionSlotNode() {
  return useContext(HeaderActionSlotNodeContext);
}
