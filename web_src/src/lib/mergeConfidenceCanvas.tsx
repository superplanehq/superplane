import type { ReactNode } from "react";

import { MergeConfidenceCanvasContext } from "./mergeConfidenceCanvasContext";

/** The merge confidence graph. Other canvases keep the saved step names. */
export function MergeConfidenceCanvasProvider({ children }: { children: ReactNode }) {
  return <MergeConfidenceCanvasContext.Provider value={true}>{children}</MergeConfidenceCanvasContext.Provider>;
}
