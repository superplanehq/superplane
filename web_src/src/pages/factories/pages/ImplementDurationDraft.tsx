import { useMemo, type ReactNode } from "react";

import { ImplementDurationDraftContext, type ImplementDurationState } from "./implementDurationState";

export function ImplementDurationDraftProvider({
  minutes,
  invalid,
  onChange,
  children,
}: ImplementDurationState & {
  onChange: (state: ImplementDurationState) => void;
  children: ReactNode;
}) {
  const value = useMemo(() => ({ minutes, invalid, setState: onChange }), [minutes, invalid, onChange]);
  return <ImplementDurationDraftContext.Provider value={value}>{children}</ImplementDurationDraftContext.Provider>;
}
