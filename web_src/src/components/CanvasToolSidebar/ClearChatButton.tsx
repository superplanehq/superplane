import { Loader2, RotateCcw } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

// Small floating action in the top-right of the agent chat (no header bar).
export function ClearChatButton({ onClearChat, clearing }: { onClearChat: () => void; clearing: boolean }) {
  return (
    <div className="absolute top-2 right-2.5 z-10">
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={onClearChat}
            disabled={clearing}
            aria-label="Clear chat"
            data-testid="agent-clear-chat-button"
            className="flex size-6 items-center justify-center rounded-md bg-white/70 text-slate-400 backdrop-blur-sm transition-colors hover:bg-slate-100 hover:text-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {clearing ? <Loader2 className="size-3.5 animate-spin" /> : <RotateCcw className="size-3.5" />}
          </button>
        </TooltipTrigger>
        <TooltipContent>Clear chat</TooltipContent>
      </Tooltip>
    </div>
  );
}
