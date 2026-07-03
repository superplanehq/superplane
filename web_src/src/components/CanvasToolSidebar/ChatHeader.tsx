import { Loader2, RotateCcw } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export function ChatHeader({ onClearChat, clearing }: { onClearChat: () => void; clearing: boolean }) {
  return (
    <div className="flex items-center justify-end border-b border-slate-200 px-2 py-1.5">
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={onClearChat}
            disabled={clearing}
            aria-label="Clear chat"
            data-testid="agent-clear-chat-button"
            className="flex size-7 items-center justify-center rounded-md text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {clearing ? <Loader2 className="size-4 animate-spin" /> : <RotateCcw className="size-4" />}
          </button>
        </TooltipTrigger>
        <TooltipContent>Clear chat</TooltipContent>
      </Tooltip>
    </div>
  );
}
