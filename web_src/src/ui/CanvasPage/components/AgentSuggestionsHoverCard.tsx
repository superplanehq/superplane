import { Button } from "@/components/ui/button";
import { Popover, PopoverAnchor, PopoverContent } from "@/ui/popover";
import { cn } from "@/lib/utils";
import { Sparkle } from "lucide-react";
import type { ReactElement } from "react";

export type AgentSuggestion = {
  id: string;
  label: string;
  /** Full prompt inserted into the Agent composer when selected. */
  prompt: string;
};

export type AgentSuggestionsHoverCardProps = {
  suggestions: AgentSuggestion[];
  children: ReactElement;
  onSelectSuggestion?: (suggestion: AgentSuggestion) => void;
  /** Controlled open state. Stays open until click-outside, Escape, or a suggestion is chosen. */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  className?: string;
};

/**
 * Wraps the Agent header control with a violet count badge and a sticky
 * suggestions card. Opens on first paint; closes only on outside click /
 * Escape. Hovering the Agent control again reopens it. Selection is handled
 * by the parent (open Agent + prefill composer + dismiss).
 */
export function AgentSuggestionsHoverCard({
  suggestions,
  children,
  onSelectSuggestion,
  open,
  onOpenChange,
  className,
}: AgentSuggestionsHoverCardProps) {
  const count = suggestions.length;

  if (count === 0) {
    return children;
  }

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverAnchor asChild>
        <div
          className={cn("relative inline-flex", className)}
          data-testid="agent-suggestions-trigger"
          onPointerEnter={() => {
            if (!open) onOpenChange?.(true);
          }}
        >
          {children}
          <span
            className="pointer-events-none absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-violet-600 px-1 text-[10px] font-semibold leading-none text-white ring-2 ring-violet-600/20 after:absolute after:inset-0 after:rounded-full after:ring-2 after:ring-white dark:bg-violet-500 dark:ring-violet-400/25 dark:after:ring-gray-900"
            aria-hidden="true"
            data-testid="agent-suggestions-badge"
          >
            {count}
          </span>
        </div>
      </PopoverAnchor>
      <PopoverContent
        align="start"
        side="bottom"
        sideOffset={6}
        className="w-[300px] p-0"
        onOpenAutoFocus={(event) => event.preventDefault()}
      >
        <div className="border-b border-violet-200/80 bg-violet-50/80 px-3 py-2.5 dark:border-violet-900/50 dark:bg-violet-950/40">
          <p className="text-[13px] font-medium text-violet-900 dark:text-violet-100">Suggested improvements</p>
          <p className="mt-0.5 text-[12px] text-violet-700/80 dark:text-violet-300/80">
            {count} {count === 1 ? "idea" : "ideas"} to set up or improve this app
          </p>
        </div>
        <ul className="flex flex-col p-1.5" role="list">
          {suggestions.map((suggestion) => (
            <li key={suggestion.id}>
              <Button
                type="button"
                variant="ghost"
                size={null}
                className="flex h-auto w-full items-center justify-start gap-2 rounded-md px-2 py-2 text-left shadow-none hover:bg-violet-50 dark:hover:bg-violet-950/50"
                data-testid={`agent-suggestion-${suggestion.id}`}
                onClick={() => onSelectSuggestion?.(suggestion)}
              >
                <span className="flex size-6 shrink-0 items-center justify-center rounded-md bg-violet-100 text-violet-700 dark:bg-violet-900/60 dark:text-violet-200">
                  <Sparkle className="size-3.5" aria-hidden="true" />
                </span>
                <span className="text-[13px] font-medium text-slate-800 dark:text-gray-100">{suggestion.label}</span>
              </Button>
            </li>
          ))}
        </ul>
        <div className="border-t border-slate-950/10 px-3 py-2 dark:border-gray-800/70">
          <p className="text-[11px] text-slate-500 dark:text-gray-400">Sends a prompt to Agent</p>
        </div>
      </PopoverContent>
    </Popover>
  );
}
