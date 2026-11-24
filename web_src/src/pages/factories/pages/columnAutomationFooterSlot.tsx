import { createContext, useContext, useEffect, type ReactNode } from "react";
import { createPortal } from "react-dom";

/** undefined: the form draws its own footer. null: the popup footer is showing a confirm step. */
const ColumnAutomationFooterSlotContext = createContext<HTMLElement | null | undefined>(undefined);
const ColumnAutomationSavingContext = createContext<((saving: boolean) => void) | undefined>(undefined);

export function ColumnAutomationFooterSlotProvider({
  slot,
  children,
  onSavingChange,
}: {
  slot: HTMLElement | null | undefined;
  children: ReactNode;
  onSavingChange?: (saving: boolean) => void;
}) {
  return (
    <ColumnAutomationSavingContext.Provider value={onSavingChange}>
      <ColumnAutomationFooterSlotContext.Provider value={slot}>{children}</ColumnAutomationFooterSlotContext.Provider>
    </ColumnAutomationSavingContext.Provider>
  );
}

function useColumnAutomationFooterSlot(): HTMLElement | null | undefined {
  return useContext(ColumnAutomationFooterSlotContext);
}

/** Renders actions on the popup footer when that footer exists, otherwise on its own row. */
export function ColumnAutomationFooterAction({ children, saving = false }: { children: ReactNode; saving?: boolean }) {
  const slot = useColumnAutomationFooterSlot();
  const onSavingChange = useContext(ColumnAutomationSavingContext);
  useEffect(() => {
    onSavingChange?.(saving);
    return () => onSavingChange?.(false);
  }, [saving, onSavingChange]);
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
