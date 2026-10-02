import { createContext, useContext, type ReactNode } from "react";
import { createPortal } from "react-dom";

/** undefined: the form draws its own footer. null: the popup footer is showing a confirm step. */
const ColumnAutomationFooterSlotContext = createContext<HTMLElement | null | undefined>(undefined);

export function ColumnAutomationFooterSlotProvider({
  slot,
  children,
}: {
  slot: HTMLElement | null | undefined;
  children: ReactNode;
}) {
  return (
    <ColumnAutomationFooterSlotContext.Provider value={slot}>{children}</ColumnAutomationFooterSlotContext.Provider>
  );
}

function useColumnAutomationFooterSlot(): HTMLElement | null | undefined {
  return useContext(ColumnAutomationFooterSlotContext);
}

/** Renders actions on the popup footer when that footer exists, otherwise on its own row. */
export function ColumnAutomationFooterAction({ children }: { children: ReactNode }) {
  const slot = useColumnAutomationFooterSlot();
  if (slot) {
    return createPortal(children, slot);
  }
  if (slot === null) {
    return null;
  }
  return (
    <footer className="flex shrink-0 items-center justify-end gap-3 border-t border-border px-5 py-3">
      {children}
    </footer>
  );
}
