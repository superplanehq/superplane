import { Button } from "@/components/ui/button";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";

/**
 * Floating pill shown over a followed log scroller once the user has
 * scrolled away from the newest output. Shared by Create-with-agent and
 * the split-run log views so scrolling behaves the same everywhere.
 */
export function JumpToLatestPill({
  onJumpToLatest,
  message = CREATE_WITH_AGENT_COPY.viewingOlder,
  action = CREATE_WITH_AGENT_COPY.jumpToLatest,
  className,
  testId = "jump-to-latest",
}: {
  onJumpToLatest: () => void;
  message?: string;
  action?: string;
  className?: string;
  testId?: string;
}) {
  return (
    <div
      className={`pointer-events-none absolute inset-x-3 bottom-3 z-10 flex justify-center ${className ?? ""}`.trim()}
      data-testid={testId}
    >
      <div className="pointer-events-auto flex items-center gap-2 rounded-full bg-zinc-900 py-1 pl-3 pr-1 text-[12px] text-white shadow-md">
        <span>{message}</span>
        <Button type="button" size="sm" className="h-7 rounded-md px-2.5 text-[12px]" onClick={onJumpToLatest}>
          {action}
        </Button>
      </div>
    </div>
  );
}
