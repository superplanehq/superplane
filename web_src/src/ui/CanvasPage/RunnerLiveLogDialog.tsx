import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useCallback, useEffect, useRef, useState } from "react";

export type RunnerLiveLogDialogProps = {
  canvasMode: "live" | "edit";
  executionId: string;
};

function useScrollToBottom(text: string) {
  const scrollRef = useRef<HTMLPreElement>(null);

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [text, scrollToBottom]);

  return { scrollRef, scrollToBottom };
}

function useLiveLogStream(executionId: string) {
  const organizationId = useOrganizationId();
  const canvasId = useCanvasId();
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);

  const { scrollRef } = useScrollToBottom(text);

  useEffect(() => {
    if (!organizationId || !canvasId || !executionId) {
      return;
    }

    const ac = new AbortController();
    setText("");
    setError(null);
    setIsStreaming(true);

    const url = `/api/v1/canvases/${encodeURIComponent(canvasId)}/node-executions/${encodeURIComponent(executionId)}/runner-live-logs`;

    (async () => {
      try {
        const res = await fetch(url, {
          method: "GET",
          credentials: "include",
          signal: ac.signal,
          ...withOrganizationHeader({
            organizationId,
            headers: { Accept: "application/x-ndjson" },
          }),
        });

        if (!res.ok) {
          const body = await res.text();
          setError(body.trim() || res.statusText || `Request failed (${res.status})`);
          setIsStreaming(false);
          return;
        }

        const reader = res.body?.getReader();
        if (!reader) {
          setError("No response body");
          setIsStreaming(false);
          return;
        }

        const decoder = new TextDecoder();
        let buffer = "";

        for (;;) {
          const { done, value } = await reader.read();
          if (done) {
            break;
          }
          buffer += decoder.decode(value, { stream: true });
          let newlineIndex: number;
          while ((newlineIndex = buffer.indexOf("\n")) >= 0) {
            const line = buffer.slice(0, newlineIndex).trim();
            buffer = buffer.slice(newlineIndex + 1);
            if (!line) {
              continue;
            }
            let rec: { type?: string; text?: string; message?: string };
            try {
              rec = JSON.parse(line) as { type?: string; text?: string; message?: string };
            } catch {
              continue;
            }
            if (rec.type === "line" && typeof rec.text === "string") {
              setText((prev) => prev + rec.text);
            } else if (rec.type === "error" && typeof rec.message === "string") {
              setError(rec.message);
            }
          }
        }
      } catch (e) {
        if ((e as Error).name === "AbortError") {
          return;
        }
        setError((e as Error).message);
      } finally {
        setIsStreaming(false);
      }
    })();

    return () => ac.abort();
  }, [organizationId, canvasId, executionId]);

  return { text, error, isStreaming, scrollRef };
}

function LiveLogStream({ executionId }: RunnerLiveLogDialogProps) {
  const { text, error, isStreaming, scrollRef } = useLiveLogStream(executionId);

  return (
    <div className="flex min-h-[50vh] flex-col overflow-hidden bg-slate-50">
      <pre
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-auto p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap text-left text-gray-800"
      >
        {error ? <span className="text-destructive">{error}</span> : null}
        {!error && !text && !isStreaming ? <span className="text-muted-foreground">No log lines yet.</span> : null}
        {!error && text}
        {!error && isStreaming && !text ? <span className="text-muted-foreground">Connecting…</span> : null}
      </pre>
    </div>
  );
}

export function RunnerLiveLogDialog({ canvasMode, executionId }: RunnerLiveLogDialogProps) {
  const organizationId = useOrganizationId();
  const canvasId = useCanvasId();
  const [open, setOpen] = useState(false);

  const canShow = canvasMode === "live" && !!organizationId && !!canvasId && !!executionId;
  if (!canShow) {
    return null;
  }

  return (
    <>
      <div className="flex justify-end border-b border-slate-950/20 px-2 py-1" data-testid="runner-live-logs">
        <Button
          type="button"
          size="sm"
          className="nodrag h-7 bg-black px-2 py-1 text-xs text-white hover:bg-black/80"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setOpen(true);
          }}
        >
          Logs
        </Button>
      </div>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          size="large"
          className="flex max-h-[min(90vh,720px)] w-[min(90vw,56rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-none"
          onClick={(e) => e.stopPropagation()}
        >
          <DialogHeader className="shrink-0 border-b border-gray-200 px-4 py-3 text-left">
            <DialogTitle>Logs</DialogTitle>
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-hidden">
            {open ? <LiveLogStream canvasMode={canvasMode} executionId={executionId} /> : null}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
