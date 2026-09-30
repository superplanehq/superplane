import { ArrowDown } from "lucide-react";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";

/**
 * Floating control over a followed log scroller once the user has
 * scrolled away from the newest output. Shared by Create-with-agent and
 * the split-run log views so scrolling behaves the same everywhere.
 */
export function JumpToLatestPill({
  onJumpToLatest,
  className,
  testId = "jump-to-latest",
}: {
  onJumpToLatest: () => void;
  className?: string;
  testId?: string;
}) {
  return (
    <div
      className={`pointer-events-none absolute inset-x-3 bottom-3 z-10 flex justify-center ${className ?? ""}`.trim()}
      data-testid={testId}
    >
      <button
        type="button"
        className="pointer-events-auto flex size-8 items-center justify-center rounded-full bg-zinc-900 text-white shadow-md"
        aria-label={CREATE_WITH_AGENT_COPY.jumpToLatest}
        onClick={onJumpToLatest}
      >
        <ArrowDown className="size-4" aria-hidden />
      </button>
    </div>
  );
}
