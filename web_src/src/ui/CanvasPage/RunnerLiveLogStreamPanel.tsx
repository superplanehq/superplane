import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useCallback, useEffect, useRef, useState } from "react";

export type RunnerLiveLogStreamPanelProps = {
  executionId: string;
};

/**
 * NDJSON runner live log stream (no outer chrome). Intended to be placed inside a dialog or other host layout.
 *
 * Organization and canvas ids are taken from the workflow URL ({@link useOrganizationId}, {@link useCanvasId}).
 */
export function RunnerLiveLogStreamPanel({ executionId }: RunnerLiveLogStreamPanelProps) {
  const organizationId = useOrganizationId();
  const canvasId = useCanvasId();
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
