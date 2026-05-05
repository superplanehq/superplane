import { createContext, type ReactNode, useContext, useState } from "react";

type SetNode = (node: ReactNode) => void;

// Two contexts so the registering side (LaunchpadView) only consumes a stable
// setter and never re-renders when the registered node changes; the rendering
// side (Header) consumes the node value. Combining both into one context with
// `{ node, setNode }` would cause the registering effect to re-fire every
// time `setNode` is called, which loops infinitely.
const LaunchpadHeaderSlotSetterContext = createContext<SetNode | null>(null);
const LaunchpadHeaderSlotNodeContext = createContext<ReactNode>(null);

export function LaunchpadHeaderSlotProvider({ children }: { children: ReactNode }) {
  const [node, setNode] = useState<ReactNode>(null);
  return (
    <LaunchpadHeaderSlotSetterContext.Provider value={setNode}>
      <LaunchpadHeaderSlotNodeContext.Provider value={node}>{children}</LaunchpadHeaderSlotNodeContext.Provider>
    </LaunchpadHeaderSlotSetterContext.Provider>
  );
}

// Returns a stable setter for the launchpad header-slot, or null when no
// provider is mounted (allowing standalone consumers to fall back gracefully).
export function useLaunchpadHeaderSlotSetter() {
  return useContext(LaunchpadHeaderSlotSetterContext);
}

// Returns the current node registered in the slot. Only the Header should read
// this — re-renders on node changes.
export function useLaunchpadHeaderSlotNode() {
  return useContext(LaunchpadHeaderSlotNodeContext);
}
