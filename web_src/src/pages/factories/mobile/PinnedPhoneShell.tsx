import type { ReactNode } from "react";

import { usePinnedPhoneShellFrame } from "./useVisualViewportFrame";

/**
 * Phone workspace shell locked to the visible screen. `bottom: 0` would
 * stick to the layout viewport and leave the bar offset after a refresh.
 */
export function PinnedPhoneShell({ children, testId }: { children: ReactNode; testId: string }) {
  const frame = usePinnedPhoneShellFrame();

  return (
    <div
      data-testid={testId}
      className="flex w-full flex-col overflow-hidden bg-background text-foreground [--workspace-navigation-width:0px]"
      style={{
        position: "fixed",
        left: 0,
        right: 0,
        top: frame.top,
        height: frame.height,
      }}
    >
      {children}
    </div>
  );
}
