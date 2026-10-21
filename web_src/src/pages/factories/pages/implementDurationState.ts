import { createContext, useContext } from "react";

export type ImplementDurationState = {
  minutes: string;
  invalid: boolean;
};

type ImplementDurationDraft = ImplementDurationState & {
  setState: (state: ImplementDurationState) => void;
};

export const ImplementDurationDraftContext = createContext<ImplementDurationDraft | null>(null);

export function useImplementDurationDraft() {
  return useContext(ImplementDurationDraftContext);
}
