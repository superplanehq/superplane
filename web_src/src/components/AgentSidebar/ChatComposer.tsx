import { Loader2, Send } from "lucide-react";
import { useCallback, useState } from "react";
// import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import type { AgentMode } from "./agentMode";
import { ModeToggle } from "./ModeToggle";

export function ChatComposer({
  onSend,
  onStop,
  sending,
  stopping,
  statusLabel: _statusLabel,
  agentMode,
  onModeSwitch,
  modeDisabled,
}: {
  onSend: (content: string) => Promise<void>;
  onStop: () => void;
  sending: boolean;
  stopping?: boolean;
  statusLabel: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  modeDisabled?: boolean;
}) {
  const [draft, setDraft] = useState("");

  const handleSend = useCallback(async () => {
    const content = draft.trim();
    if (!content || sending) return;

    setDraft("");

    try {
      await onSend(content);
    } catch {
      setDraft((currentDraft) => (currentDraft.trim() ? currentDraft : content));
    }
  }, [draft, onSend, sending]);

  return (
    <footer className="border-t border-border p-3 flex flex-col gap-2">
      <Textarea
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        rows={3}
        placeholder="Ask the agent…"
        data-testid="agent-input"
        className="resize-none"
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            if (draft.trim()) {
              void handleSend();
            }
          }
        }}
      />
      <div className="flex items-center justify-between">
        <ModeToggle mode={agentMode} onSwitch={onModeSwitch} disabled={modeDisabled} streaming={sending} />
        <div className="flex items-center gap-2">
          {sending && (
            <Button
              type="button"
              variant="destructive"
              size="icon"
              onClick={onStop}
              disabled={stopping}
              data-testid="agent-stop-button"
              title={stopping ? "Stopping..." : "Stop"}
            >
              {stopping ? (
                <Loader2 className="size-3 animate-spin" />
              ) : (
                <div className="size-3 rounded-sm bg-white" />
              )}
            </Button>
          )}
          <Button
            type="button"
            onClick={() => void handleSend()}
            disabled={!draft.trim()}
            data-testid="agent-send-message-button"
          >
            <Send className="size-4" />
            Send
          </Button>
        </div>
      </div>
    </footer>
  );
}
