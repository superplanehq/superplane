import { cn } from "@/lib/utils";

import { WORKSPACE_LOADING_TEST_ID } from "@/lib/workspaceLoadingCopy";

import { LOADING_REVEAL_CLASSNAME } from "../lib/loadingReveal";

const WORKSPACE_LOADING_STROKES = [
  { name: "left", d: "M14.4975 0 L3.123 16.447" },
  { name: "right", d: "M14.4975 0 L25.877 16.455" },
  { name: "stem", d: "M14.4975 0 L14.5 20.3" },
] as const;

function WorkspaceLoadingMark() {
  return (
    <div className="workspace-loading-mark" aria-hidden>
      <svg viewBox="-2 -2 33 25" className="workspace-loading-trace-svg">
        {WORKSPACE_LOADING_STROKES.map((stroke) => (
          <path
            key={stroke.name}
            className={`workspace-loading-trace workspace-loading-trace--${stroke.name}`}
            d={stroke.d}
            pathLength="1"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.85"
            strokeLinecap="round"
          />
        ))}
      </svg>
      <span className="workspace-loading-pen" />
    </div>
  );
}

export function WorkspaceLoadingScreen({ message }: { message: string }) {
  return (
    <div
      className="fixed inset-0 z-50 flex min-h-screen flex-col items-center justify-center gap-5 bg-background text-foreground"
      role="status"
      aria-label={message}
      aria-live="polite"
      aria-busy="true"
      data-testid={WORKSPACE_LOADING_TEST_ID}
    >
      <WorkspaceLoadingMark />
      <p key={message} className={cn("text-[15px] font-medium text-foreground", LOADING_REVEAL_CLASSNAME)}>
        {message}
      </p>
    </div>
  );
}
