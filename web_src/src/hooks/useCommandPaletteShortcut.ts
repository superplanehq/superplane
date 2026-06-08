import { useEffect, useState } from "react";

const isEditableTarget = (target: EventTarget | null): boolean => {
  if (!(target instanceof Element)) return false;
  return !!target.closest('input, textarea, select, [contenteditable="true"], .monaco-editor');
};

export interface UseCommandPaletteShortcutResult {
  open: boolean;
  setOpen: (open: boolean) => void;
}

export function useCommandPaletteShortcut(): UseCommandPaletteShortcutResult {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "k" || !(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey) {
        return;
      }
      if (!open && isEditableTarget(event.target)) {
        return;
      }
      event.preventDefault();
      setOpen((prev) => !prev);
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open]);

  return { open, setOpen };
}
