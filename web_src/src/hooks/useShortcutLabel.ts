import { formatShortcutLabel, getShortcutModifierLabel } from "@/lib/keyboardShortcuts";
import { useEffect, useState } from "react";

export function useShortcutModifierLabel() {
  const [modifier, setModifier] = useState(getShortcutModifierLabel);

  useEffect(() => {
    setModifier(getShortcutModifierLabel());
  }, []);

  return modifier;
}

export function useShortcutLabel(key: string) {
  const [label, setLabel] = useState(() => formatShortcutLabel(key));

  useEffect(() => {
    setLabel(formatShortcutLabel(key));
  }, [key]);

  return label;
}
