import { useEffect } from "react";

export type UseApplyOnCmdEnterParams = {
  /** When false, no key listener is registered. */
  enabled: boolean;
  disabled: boolean;
  isApplying: boolean;
  onApply: () => void | Promise<void>;
};

export function useApplyOnCmdEnter({ enabled, disabled, isApplying, onApply }: UseApplyOnCmdEnterParams): void {
  useEffect(() => {
    if (!enabled) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.isComposing || event.key !== "Enter") {
        return;
      }

      if (!(event.metaKey || event.ctrlKey)) {
        return;
      }

      if (disabled || isApplying) {
        return;
      }

      event.preventDefault();
      void onApply();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [disabled, enabled, isApplying, onApply]);
}
