import { createContext, useContext } from "react";

export const MergeConfidenceCanvasContext = createContext(false);

export function useMergeConfidenceCanvas(): boolean {
  return useContext(MergeConfidenceCanvasContext);
}
