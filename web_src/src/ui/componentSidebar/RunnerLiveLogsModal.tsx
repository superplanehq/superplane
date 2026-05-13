import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useCallback, useEffect, useRef, useState } from "react";

type RunnerLiveLogsModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  canvasId: string;
  executionId: string;
};

export function RunnerLiveLogsModal({
  open,
  onOpenChange,
  organizationId,
  canvasId,
  executionId,
}: RunnerLiveLogsModalProps) {
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);
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

  useEffect(() => {
    if (!open || !organizationId || !canvasId || !executionId) {
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
          ...withOrganizationHeader({ organizationId }),
          headers: { Accept: "application/x-ndjson" },
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
  }, [open, organizationId, canvasId, executionId]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl gap-0 p-0">
        <DialogHeader className="px-6 pt-6 pb-2">
          <DialogTitle>Runner live logs</DialogTitle>
          <DialogDescription>
            Streams stdout and stderr from the task broker&apos;s CloudWatch log sink when it is configured for your
            fleet.
          </DialogDescription>
        </DialogHeader>
        <div className="px-6 pb-2">
          <pre
            ref={scrollRef}
            className="h-[min(70vh,420px)] w-full overflow-auto rounded-md border border-border bg-muted/40 p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap text-left"
          >
            {error ? <span className="text-destructive">{error}</span> : null}
            {!error && !text && !isStreaming ? <span className="text-muted-foreground">No log lines yet.</span> : null}
            {!error && text}
            {!error && isStreaming && !text ? <span className="text-muted-foreground">Connecting…</span> : null}
          </pre>
        </div>
        <DialogFooter className="px-6 pb-6">
          <Button type="button" variant="secondary" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
