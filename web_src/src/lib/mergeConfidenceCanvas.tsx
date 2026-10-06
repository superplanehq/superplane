import { createContext, useContext, type ReactNode } from "react";

const MergeConfidenceCanvasContext = createContext(false);

/** The merge confidence graph. Other canvases keep the saved step names. */
export function MergeConfidenceCanvasProvider({ children }: { children: ReactNode }) {
  return <MergeConfidenceCanvasContext.Provider value={true}>{children}</MergeConfidenceCanvasContext.Provider>;
}

export function useMergeConfidenceCanvas(): boolean {
  return useContext(MergeConfidenceCanvasContext);
}
