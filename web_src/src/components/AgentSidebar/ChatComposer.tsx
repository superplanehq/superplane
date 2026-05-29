import { ArrowUp, Loader2, Square } from "lucide-react";
import { useCallback, useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import type { AgentMode } from "./agentMode";
import { ModeToggle } from "./ModeToggle";

export function ChatComposer({
  onSend,
  onStop,
  sending,
  sendPending,
  stopping,
  statusLabel,
  agentMode,
  onModeSwitch,
  modeDisabled,
}: {
  onSend: (content: string) => Promise<void>;
  onStop: () => void;
  sending: boolean;
  sendPending: boolean;
  stopping?: boolean;
  statusLabel: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  modeDisabled?: boolean;
}) {
  const [draft, setDraft] = useState("");
  const canSend = Boolean(draft.trim()) && !sendPending;

  const handleSend = useCallback(async () => {
    const content = draft.trim();
    if (!content) return;

    setDraft("");

    try {
      await onSend(content);
    } catch {
      setDraft((currentDraft) => (currentDraft.trim() ? currentDraft : content));
    }
  }, [draft, onSend]);

  return (
    <footer className="px-3 pb-3 pt-2">
      <div className="mx-auto w-full max-w-[800px] overflow-hidden rounded-lg bg-white shadow-sm outline outline-1 outline-slate-950/15">
        <Textarea
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          rows={1}
          placeholder="Ask the agent…"
          data-testid="agent-input"
          className={cn(
            "min-h-9 w-full resize-none border-0 bg-transparent px-3 py-2.5 text-sm shadow-none",
            "outline-none ring-0 focus-visible:border-0 focus-visible:ring-0 focus-visible:outline-none",
            "placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50",
            "text-[rgba(10,10,10,1)] dark:bg-transparent",
          )}
          onKeyDown={(e) => {
            if (e.key !== "Enter") return;
            const nativeEvent = e.nativeEvent;
            if ("isComposing" in nativeEvent && nativeEvent.isComposing) return;
            if (e.shiftKey) return;
            e.preventDefault();
            if (!canSend) return;
            void handleSend();
          }}
        />
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
              onClick={() => void handleSend()}
              disabled={!canSend}
              aria-label="Send message"
              data-testid="agent-send-message-button"
            >
              <ArrowUp className="size-3.5" aria-hidden />
            </Button>
          </div>
        </div>
      </div>
    </footer>
  );
}
