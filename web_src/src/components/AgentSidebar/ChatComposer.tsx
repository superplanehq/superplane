import { Loader2, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import type { AgentMode } from "./useAgentState";
import { ModeToggle } from "./ModeToggle";

export function ChatComposer({
  draft,
  onDraftChange,
  onSend,
  onStop,
  sending,
  stopping,
  statusLabel,
  agentMode,
  onModeSwitch,
  modeDisabled,
}: {
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  onStop: () => void;
  sending: boolean;
  stopping?: boolean;
  statusLabel: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  modeDisabled?: boolean;
}) {
  return (
    <footer className="border-t border-border p-3 flex flex-col gap-2">
      <Textarea
        value={draft}
        onChange={(e) => onDraftChange(e.target.value)}
        rows={3}
        placeholder="Ask the agent…"
        data-testid="agent-input"
        className="resize-none"
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            if (!sending) onSend();
          }
        }}
      />
      <div className="flex items-center justify-between">
        <ModeToggle mode={agentMode} onSwitch={onModeSwitch} disabled={modeDisabled} streaming={sending} />
        {sending ? (
          <Button
            type="button"
            variant="destructive"
            onClick={onStop}
            disabled={stopping}
            data-testid="agent-stop-button"
            className="gap-1"
          >
            {stopping ? (
              <Loader2 className="size-3 animate-spin" />
            ) : (
              <div className="size-3 rounded-sm bg-white animate-pulse" />
            )}
            {stopping ? "Stopping..." : "Stop"}
          </Button>
        ) : (
          <Button type="button" onClick={onSend} disabled={!draft.trim()} data-testid="agent-send-message-button">
            <Send className="size-4" />
            Send
          </Button>
        )}
      </div>
    </footer>
  );
}
