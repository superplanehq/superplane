import { memo } from "react";
import { ArrowUp, Loader2, Square } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AgentMode } from "./agentMode";
import { ModeToggle } from "./ModeToggle";

interface ComposerToolbarProps {
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  modeDisabled?: boolean;
  sending: boolean;
  stopping?: boolean;
  statusLabel: string;
  canSend: boolean;
  onStop: () => void;
  onSend: () => void;
}

export const ComposerToolbar = memo(function ComposerToolbar({
  agentMode,
  onModeSwitch,
  modeDisabled,
  sending,
  stopping,
  statusLabel,
  canSend,
  onStop,
  onSend,
}: ComposerToolbarProps) {
  return (
    <div className="flex items-center justify-between gap-2 px-2 pb-2">
      <ModeToggle mode={agentMode} onSwitch={onModeSwitch} disabled={modeDisabled} streaming={sending} />
      <div className="flex min-w-0 shrink-0 items-center gap-2">
        <span className="truncate text-xs text-muted-foreground">{statusLabel}</span>
        {sending && (
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="size-7 shrink-0 rounded-full border-slate-300 bg-white text-slate-700 hover:bg-slate-100"
            onClick={onStop}
            disabled={stopping}
            aria-label={stopping ? "Stopping" : "Stop"}
            title={stopping ? "Stopping..." : "Stop"}
            data-testid="agent-stop-button"
          >
            {stopping ? (
              <Loader2 className="size-3.5 animate-spin" aria-hidden />
            ) : (
              <Square className="size-3 fill-current" aria-hidden />
            )}
          </Button>
        )}
        <Button
          type="button"
          variant="default"
          size="icon"
          className="size-7 shrink-0 rounded-full"
          onClick={onSend}
          disabled={!canSend}
          aria-label="Send message"
          data-testid="agent-send-message-button"
        >
          <ArrowUp className="size-3.5" aria-hidden />
        </Button>
      </div>
    </div>
  );
});
